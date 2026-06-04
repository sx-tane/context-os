package tests

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/entities"
	"context-os/domain/pipelines"
	"context-os/domain/types"
	"context-os/internal/pipeline"
	githubsource "context-os/internal/source/github"
	"context-os/internal/stages/ingestion"

	"gopkg.in/yaml.v2"
)

type pipelineScenario struct {
	ID          string `yaml:"id"`
	Area        string `yaml:"area"`
	Level       string `yaml:"level"`
	Description string `yaml:"description"`
	Owner       string `yaml:"owner"`
	Inputs      struct {
		SourceRequest struct {
			URI          string `yaml:"uri"`
			ContentPath  string `yaml:"content_path"`
			MetadataPath string `yaml:"metadata_path"`
		} `yaml:"source_request"`
		Assistant struct {
			ImprovesRecall bool                    `yaml:"improves_recall"`
			Proposals      []assistantRelationship `yaml:"proposals"`
		} `yaml:"assistant"`
	} `yaml:"inputs"`
	Expected struct {
		GoldenPath string `yaml:"golden_path"`
	} `yaml:"expected"`
	Thresholds metricThresholds `yaml:"thresholds"`
	Evidence   []string         `yaml:"evidence"`
}

type metricThresholds struct {
	PrecisionMin                     float64 `yaml:"precision_min"`
	RecallMin                        float64 `yaml:"recall_min"`
	FalsePositiveRateMax             float64 `yaml:"false_positive_rate_max"`
	RelationshipPrecisionMin         float64 `yaml:"relationship_precision_min"`
	RelationshipRecallMin            float64 `yaml:"relationship_recall_min"`
	RelationshipFalsePositiveRateMax float64 `yaml:"relationship_false_positive_rate_max"`
}

type pipelineGolden struct {
	Entities      []goldenEntity       `json:"entities"`
	Relationships []goldenRelationship `json:"relationships"`
	Mismatches    []goldenMismatch     `json:"mismatches"`
}

type goldenEntity struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type goldenMismatch struct {
	Type          string   `json:"type"`
	Summary       string   `json:"summary"`
	Severity      string   `json:"severity"`
	ConfidenceMin float64  `json:"confidence_min"`
	Impact        string   `json:"impact"`
	Evidence      []string `json:"evidence"`
	EntityNames   []string `json:"entity_names"`
}

type goldenRelationship struct {
	From          string   `json:"from"`
	To            string   `json:"to"`
	Kind          string   `json:"kind"`
	ConfidenceMin float64  `json:"confidence_min"`
	Evidence      []string `json:"evidence"`
}

type assistantRelationship struct {
	From       string  `yaml:"from"`
	To         string  `yaml:"to"`
	Kind       string  `yaml:"kind"`
	Evidence   string  `yaml:"evidence"`
	Confidence float64 `yaml:"confidence"`
}

type scenarioRelationshipAssistant struct {
	proposals []assistantRelationship
}

// ProposeRelationships converts scenario relationship proposals into fake assistant edges.
func (a scenarioRelationshipAssistant) ProposeRelationships(ctx context.Context, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) ([]types.Relationship, error) {
	_ = ctx
	_ = doc
	byName := canonicalByName(canonical)
	out := make([]types.Relationship, 0, len(a.proposals))
	for _, proposal := range a.proposals {
		from, fromOK := byName[proposal.From]
		to, toOK := byName[proposal.To]
		if !fromOK || !toOK {
			continue
		}
		out = append(out, types.Relationship{
			FromID:     from.Entity.ID,
			ToID:       to.Entity.ID,
			Kind:       types.RelationshipKind(proposal.Kind),
			Confidence: proposal.Confidence,
			Evidence:   []string{proposal.Evidence},
		})
	}
	return out, nil
}

// Provider returns the fake provider label used in benchmark metadata.
func (a scenarioRelationshipAssistant) Provider() string {
	return "fake_benchmark"
}

// TestPipelineHarnessScenarios verifies shared pipeline scenarios produce deterministic semantic entities and evidence-backed mismatches.
func TestPipelineHarnessScenarios(t *testing.T) {
	scenarios := loadPipelineScenarios(t)
	if len(scenarios) == 0 {
		t.Fatalf("loadPipelineScenarios() length = %d, want at least one scenario", len(scenarios))
	}

	for _, scenario := range scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			assertScenarioContract(t, scenario)

			golden := loadPipelineGolden(t, scenario.Expected.GoldenPath)
			result := runPipelineScenario(t, scenario, nil)
			sortPipelineResult(&result)

			matchedEntities := assertGoldenEntities(t, result.Entities, golden.Entities)
			matchedMismatches := assertGoldenMismatches(t, result, golden.Mismatches)
			assertMetricThresholds(t, scenario.Thresholds, matchedEntities, len(golden.Entities), matchedMismatches, len(golden.Mismatches), len(result.Mismatches))
		})
	}
}

// TestRelationshipBenchmarkScenarios verifies relationship quality gates for deterministic baseline and fake-assisted runs.
func TestRelationshipBenchmarkScenarios(t *testing.T) {
	scenarios := loadRelationshipScenarios(t)
	if len(scenarios) == 0 {
		t.Fatalf("loadRelationshipScenarios() length = %d, want at least one scenario", len(scenarios))
	}

	for _, scenario := range scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			assertRelationshipScenarioContract(t, scenario)

			golden := loadPipelineGolden(t, scenario.Expected.GoldenPath)
			baseline := runPipelineScenario(t, scenario, nil)
			sortPipelineResult(&baseline)

			assisted := runPipelineScenario(t, scenario, &pipeline.Stores{
				RelationshipAssistant: scenarioRelationshipAssistant{proposals: scenario.Inputs.Assistant.Proposals},
			})
			sortPipelineResult(&assisted)

			assertGoldenEntities(t, assisted.Entities, golden.Entities)
			assertGoldenMismatches(t, assisted, golden.Mismatches)
			metrics := relationshipMetricResult(assisted, golden.Relationships)
			assertRelationshipThresholds(t, scenario.Thresholds, metrics)
			assertGoldenRelationships(t, assisted, golden.Relationships)

			if scenario.Inputs.Assistant.ImprovesRecall {
				baselineMetrics := relationshipMetricResult(baseline, golden.Relationships)
				if metrics.Recall <= baselineMetrics.Recall {
					t.Fatalf("relationship recall = %v, want greater than deterministic baseline %v", metrics.Recall, baselineMetrics.Recall)
				}
			}
		})
	}
}

// TestRelationshipMetricsDetectMissingExpectedEdges verifies recall drops below the benchmark gate when expected edges are absent.
func TestRelationshipMetricsDetectMissingExpectedEdges(t *testing.T) {
	result := pipelines.Result{}
	want := []goldenRelationship{{From: "A", To: "B", Kind: string(types.RequirementAffectsAPI)}}

	metrics := relationshipMetricResult(result, want)

	if metrics.Recall >= 0.70 {
		t.Fatalf("relationship recall = %v, want below missing-edge gate", metrics.Recall)
	}
}

// TestRelationshipMetricsDetectUnsupportedExtraEdges verifies false-positive rate rises when unsupported edges are emitted.
func TestRelationshipMetricsDetectUnsupportedExtraEdges(t *testing.T) {
	result := pipelines.Result{
		Entities: []entities.CanonicalEntity{
			{Entity: types.Entity{ID: "a", Name: "A"}},
			{Entity: types.Entity{ID: "b", Name: "B"}},
		},
		Relationships: []types.Relationship{{
			ID:     "a->b:requirement_affects_api",
			FromID: "a",
			ToID:   "b",
			Kind:   types.RequirementAffectsAPI,
		}},
	}

	metrics := relationshipMetricResult(result, nil)

	if metrics.FalsePositiveRate <= 0.15 {
		t.Fatalf("relationship false positive rate = %v, want above extra-edge gate", metrics.FalsePositiveRate)
	}
}

// loadPipelineScenarios returns every shared reasoning pipeline scenario in deterministic path order.
func loadPipelineScenarios(t *testing.T) []pipelineScenario {
	t.Helper()

	paths, err := filepath.Glob(repoPath(t, "tests/harness/scenarios/reasoning/*.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	sort.Strings(paths)

	scenarios := make([]pipelineScenario, 0, len(paths))
	for _, path := range paths {
		var scenario pipelineScenario
		if err := yaml.Unmarshal(readFile(t, path), &scenario); err != nil {
			t.Fatalf("Unmarshal(%q) error = %v", path, err)
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

// loadRelationshipScenarios returns every shared relationship benchmark scenario in deterministic path order.
func loadRelationshipScenarios(t *testing.T) []pipelineScenario {
	t.Helper()

	paths, err := filepath.Glob(repoPath(t, "tests/harness/scenarios/relationship/*.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	sort.Strings(paths)

	scenarios := make([]pipelineScenario, 0, len(paths))
	for _, path := range paths {
		var scenario pipelineScenario
		if err := yaml.Unmarshal(readFile(t, path), &scenario); err != nil {
			t.Fatalf("Unmarshal(%q) error = %v", path, err)
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

// assertScenarioContract verifies required harness scenario fields are populated with expected scope values.
func assertScenarioContract(t *testing.T, scenario pipelineScenario) {
	t.Helper()

	if scenario.ID == "" {
		t.Fatalf("scenario ID = %q, want stable identifier", scenario.ID)
	}
	if scenario.Area != "reasoning" {
		t.Fatalf("scenario area = %q, want reasoning", scenario.Area)
	}
	if scenario.Level != "pipeline" {
		t.Fatalf("scenario level = %q, want pipeline", scenario.Level)
	}
	if scenario.Description == "" {
		t.Fatalf("scenario description = %q, want one sentence", scenario.Description)
	}
	if scenario.Owner != "internal/pipeline" {
		t.Fatalf("scenario owner = %q, want internal/pipeline", scenario.Owner)
	}
	if scenario.Inputs.SourceRequest.URI == "" {
		t.Fatalf("scenario source request URI = %q, want fixture URI", scenario.Inputs.SourceRequest.URI)
	}
	if scenario.Inputs.SourceRequest.ContentPath == "" {
		t.Fatalf("scenario content path = %q, want fixture path", scenario.Inputs.SourceRequest.ContentPath)
	}
	if scenario.Expected.GoldenPath == "" {
		t.Fatalf("scenario golden path = %q, want golden output path", scenario.Expected.GoldenPath)
	}
	if len(scenario.Evidence) == 0 {
		t.Fatalf("scenario evidence length = %d, want fixture evidence reference", len(scenario.Evidence))
	}
}

// assertRelationshipScenarioContract verifies required relationship benchmark fields are populated.
func assertRelationshipScenarioContract(t *testing.T, scenario pipelineScenario) {
	t.Helper()

	if scenario.ID == "" {
		t.Fatalf("scenario ID = %q, want stable identifier", scenario.ID)
	}
	if scenario.Area != "relationship" {
		t.Fatalf("scenario area = %q, want relationship", scenario.Area)
	}
	if scenario.Level != "benchmark" {
		t.Fatalf("scenario level = %q, want benchmark", scenario.Level)
	}
	if scenario.Description == "" {
		t.Fatalf("scenario description = %q, want one sentence", scenario.Description)
	}
	if scenario.Owner != "internal/pipeline" {
		t.Fatalf("scenario owner = %q, want internal/pipeline", scenario.Owner)
	}
	if scenario.Inputs.SourceRequest.URI == "" {
		t.Fatalf("scenario source request URI = %q, want fixture URI", scenario.Inputs.SourceRequest.URI)
	}
	if scenario.Inputs.SourceRequest.ContentPath == "" {
		t.Fatalf("scenario content path = %q, want fixture path", scenario.Inputs.SourceRequest.ContentPath)
	}
	if scenario.Expected.GoldenPath == "" {
		t.Fatalf("scenario golden path = %q, want golden output path", scenario.Expected.GoldenPath)
	}
	if scenario.Thresholds.RelationshipPrecisionMin == 0 {
		t.Fatalf("relationship precision threshold = %v, want configured gate", scenario.Thresholds.RelationshipPrecisionMin)
	}
	if scenario.Thresholds.RelationshipRecallMin == 0 {
		t.Fatalf("relationship recall threshold = %v, want configured gate", scenario.Thresholds.RelationshipRecallMin)
	}
	if len(scenario.Evidence) == 0 {
		t.Fatalf("scenario evidence length = %d, want fixture evidence reference", len(scenario.Evidence))
	}
}

// loadPipelineGolden reads the deterministic semantic golden output for one scenario.
func loadPipelineGolden(t *testing.T, path string) pipelineGolden {
	t.Helper()

	var golden pipelineGolden
	if err := json.Unmarshal(readRepoFile(t, path), &golden); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	return golden
}

// runPipelineScenario executes one scenario through the current full pipeline with local fixtures only.
func runPipelineScenario(t *testing.T, scenario pipelineScenario, stores *pipeline.Stores) pipelines.Result {
	t.Helper()

	metadata := map[string]string{}
	metadataPath := scenario.Inputs.SourceRequest.MetadataPath
	if metadataPath != "" {
		if err := json.Unmarshal(readRepoFile(t, metadataPath), &metadata); err != nil {
			t.Fatalf("Unmarshal(%q) error = %v", metadataPath, err)
		}
	}
	content := string(readRepoFile(t, scenario.Inputs.SourceRequest.ContentPath))
	pipe := ingestion.NewPipeline(githubsource.NewConnector())
	result, err := pipeline.Run(context.Background(), pipe, contracts.SourceRequest{
		URI:      scenario.Inputs.SourceRequest.URI,
		Content:  content,
		Metadata: metadata,
	}, stores)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return result
}

// sortPipelineResult makes entity and mismatch ordering stable before assertions.
func sortPipelineResult(result *pipelines.Result) {
	sort.Slice(result.Entities, func(i, j int) bool {
		return result.Entities[i].Entity.ID < result.Entities[j].Entity.ID
	})
	sort.Slice(result.Mismatches, func(i, j int) bool {
		return result.Mismatches[i].ID < result.Mismatches[j].ID
	})
	sort.Slice(result.Relationships, func(i, j int) bool {
		return result.Relationships[i].ID < result.Relationships[j].ID
	})
}

// assertGoldenEntities verifies each semantic golden entity exists in the pipeline output.
func assertGoldenEntities(t *testing.T, got []entities.CanonicalEntity, want []goldenEntity) int {
	t.Helper()

	matched := 0
	for _, expected := range want {
		entity, ok := findEntity(got, expected.Name)
		if !ok {
			t.Fatalf("entity %q present = false, want true", expected.Name)
		}
		matched++
		if string(entity.Entity.Type) != expected.Type {
			t.Fatalf("entity %q type = %q, want %q", expected.Name, entity.Entity.Type, expected.Type)
		}
		if entity.Entity.SourceID == "" {
			t.Fatalf("entity %q SourceID = empty, want provenance", expected.Name)
		}
	}
	return matched
}

// assertGoldenMismatches verifies each expected mismatch includes confidence, impact, evidence, and entity references.
func assertGoldenMismatches(t *testing.T, result pipelines.Result, want []goldenMismatch) int {
	t.Helper()

	if len(want) == 0 {
		if len(result.Mismatches) != 0 {
			t.Fatalf("Mismatches length = %d, want 0", len(result.Mismatches))
		}
		return 0
	}

	entityNames := entityNamesByID(result.Entities)
	matched := 0
	for _, expected := range want {
		mismatch, ok := findMismatch(result.Mismatches, expected.Summary)
		if !ok {
			t.Fatalf("mismatch %q present = false, want true", expected.Summary)
		}
		matched++
		if mismatch.Type != expected.Type {
			t.Fatalf("mismatch %q type = %q, want %q", expected.Summary, mismatch.Type, expected.Type)
		}
		if mismatch.Severity != expected.Severity {
			t.Fatalf("mismatch %q severity = %q, want %q", expected.Summary, mismatch.Severity, expected.Severity)
		}
		if mismatch.Confidence < expected.ConfidenceMin {
			t.Fatalf("mismatch %q confidence = %v, want >= %v", expected.Summary, mismatch.Confidence, expected.ConfidenceMin)
		}
		if mismatch.Impact != expected.Impact {
			t.Fatalf("mismatch %q impact = %q, want %q", expected.Summary, mismatch.Impact, expected.Impact)
		}
		if !sameStrings(mismatch.Evidence, expected.Evidence) {
			t.Fatalf("mismatch %q evidence = %v, want %v", expected.Summary, mismatch.Evidence, expected.Evidence)
		}
		gotNames := mismatchEntityNames(mismatch, entityNames)
		if !sameStrings(gotNames, expected.EntityNames) {
			t.Fatalf("mismatch %q entity names = %v, want %v", expected.Summary, gotNames, expected.EntityNames)
		}
	}
	return matched
}

// assertGoldenRelationships verifies each expected semantic relationship includes confidence and evidence.
func assertGoldenRelationships(t *testing.T, result pipelines.Result, want []goldenRelationship) int {
	t.Helper()

	entityNames := entityNamesByID(result.Entities)
	matched := 0
	for _, expected := range want {
		rel, ok := findGoldenRelationship(result.Relationships, entityNames, expected)
		if !ok {
			t.Fatalf("relationship %s -> %s (%s) present = false, want true", expected.From, expected.To, expected.Kind)
		}
		matched++
		if rel.Confidence < expected.ConfidenceMin {
			t.Fatalf("relationship %s -> %s confidence = %v, want >= %v", expected.From, expected.To, rel.Confidence, expected.ConfidenceMin)
		}
		if !sameStrings(rel.Evidence, expected.Evidence) {
			t.Fatalf("relationship %s -> %s evidence = %v, want %v", expected.From, expected.To, rel.Evidence, expected.Evidence)
		}
	}
	return matched
}

type relationshipMetrics struct {
	Precision         float64
	Recall            float64
	FalsePositiveRate float64
	Matched           int
	Expected          int
	Actual            int
}

// relationshipMetricResult computes semantic relationship precision, recall, and false-positive rate.
func relationshipMetricResult(result pipelines.Result, want []goldenRelationship) relationshipMetrics {
	entityNames := entityNamesByID(result.Entities)
	actual := semanticRelationships(result.Relationships)
	matched := 0
	for _, expected := range want {
		if _, ok := findGoldenRelationship(actual, entityNames, expected); ok {
			matched++
		}
	}
	return relationshipMetrics{
		Precision:         precision(matched, len(actual)),
		Recall:            ratio(matched, len(want)),
		FalsePositiveRate: falsePositiveRate(matched, len(actual)),
		Matched:           matched,
		Expected:          len(want),
		Actual:            len(actual),
	}
}

// assertRelationshipThresholds verifies relationship precision, recall, and false-positive gates pass.
func assertRelationshipThresholds(t *testing.T, thresholds metricThresholds, metrics relationshipMetrics) {
	t.Helper()

	if metrics.Precision < thresholds.RelationshipPrecisionMin {
		t.Fatalf("relationship precision = %v, want >= %v", metrics.Precision, thresholds.RelationshipPrecisionMin)
	}
	if metrics.Recall < thresholds.RelationshipRecallMin {
		t.Fatalf("relationship recall = %v, want >= %v", metrics.Recall, thresholds.RelationshipRecallMin)
	}
	if metrics.FalsePositiveRate > thresholds.RelationshipFalsePositiveRateMax {
		t.Fatalf("relationship false positive rate = %v, want <= %v", metrics.FalsePositiveRate, thresholds.RelationshipFalsePositiveRateMax)
	}
}

// assertMetricThresholds verifies scenario precision, recall, and false-positive gates pass for this run.
func assertMetricThresholds(t *testing.T, thresholds metricThresholds, matchedEntities, expectedEntities, matchedMismatches, expectedMismatches, actualMismatches int) {
	t.Helper()

	entityRecall := ratio(matchedEntities, expectedEntities)
	if entityRecall < thresholds.RecallMin {
		t.Fatalf("entity recall = %v, want >= %v", entityRecall, thresholds.RecallMin)
	}

	mismatchRecall := ratio(matchedMismatches, expectedMismatches)
	if mismatchRecall < thresholds.RecallMin {
		t.Fatalf("mismatch recall = %v, want >= %v", mismatchRecall, thresholds.RecallMin)
	}

	precision := precision(matchedMismatches, actualMismatches)
	if precision < thresholds.PrecisionMin {
		t.Fatalf("precision = %v, want >= %v", precision, thresholds.PrecisionMin)
	}

	falsePositiveRate := falsePositiveRate(matchedMismatches, actualMismatches)
	if falsePositiveRate > thresholds.FalsePositiveRateMax {
		t.Fatalf("false positive rate = %v, want <= %v", falsePositiveRate, thresholds.FalsePositiveRateMax)
	}
}

// repoRoot returns the repository root path for tests run from either the repo root or tests package directory.
func repoRoot(t *testing.T) string {
	t.Helper()

	if _, err := os.Stat("tests/harness/README.md"); err == nil {
		return "."
	}
	if _, err := os.Stat("harness/README.md"); err == nil {
		return ".."
	}
	t.Fatalf("repo root = %q, want repository root or tests package directory", "")
	return ""
}

// repoPath converts a repo-relative harness path into a path usable from the current test working directory.
func repoPath(t *testing.T, path string) string {
	t.Helper()

	return filepath.Join(repoRoot(t), path)
}

// readRepoFile reads a repo-relative fixture, scenario, or golden file.
func readRepoFile(t *testing.T, path string) []byte {
	t.Helper()

	return readFile(t, repoPath(t, path))
}

// readFile reads a file from an already resolved path.
func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}

// findEntity returns the entity whose surface name matches name.
func findEntity(input []entities.CanonicalEntity, name string) (entities.CanonicalEntity, bool) {
	for _, entity := range input {
		if entity.Entity.Name == name {
			return entity, true
		}
	}
	return entities.CanonicalEntity{}, false
}

// findMismatch returns the mismatch whose summary matches summary.
func findMismatch(mismatches []types.Mismatch, summary string) (types.Mismatch, bool) {
	for _, mismatch := range mismatches {
		if mismatch.Summary == summary {
			return mismatch, true
		}
	}
	return types.Mismatch{}, false
}

// findGoldenRelationship returns the relationship matching the expected names and kind.
func findGoldenRelationship(rels []types.Relationship, names map[string]string, expected goldenRelationship) (types.Relationship, bool) {
	for _, rel := range rels {
		if rel.Kind == types.CoOccursInDocument {
			continue
		}
		if names[rel.FromID] != expected.From {
			continue
		}
		if names[rel.ToID] != expected.To {
			continue
		}
		if string(rel.Kind) != expected.Kind {
			continue
		}
		if len(expected.Evidence) > 0 && !sameStrings(rel.Evidence, expected.Evidence) {
			continue
		}
		return rel, true
	}
	return types.Relationship{}, false
}

// semanticRelationships returns non-co-occurrence relationships for quality scoring.
func semanticRelationships(rels []types.Relationship) []types.Relationship {
	out := make([]types.Relationship, 0, len(rels))
	for _, rel := range rels {
		if rel.Kind == types.CoOccursInDocument {
			continue
		}
		out = append(out, rel)
	}
	return out
}

// entityNamesByID returns a lookup table from entity ID to entity surface name.
func entityNamesByID(input []entities.CanonicalEntity) map[string]string {
	out := make(map[string]string, len(input))
	for _, entity := range input {
		out[entity.Entity.ID] = entity.Entity.Name
	}
	return out
}

// canonicalByName returns a lookup table from entity surface name to canonical entity.
func canonicalByName(input []entities.CanonicalEntity) map[string]entities.CanonicalEntity {
	out := make(map[string]entities.CanonicalEntity, len(input))
	for _, entity := range input {
		out[entity.Entity.Name] = entity
	}
	return out
}

// mismatchEntityNames resolves mismatch entity IDs to sorted surface names.
func mismatchEntityNames(mismatch types.Mismatch, names map[string]string) []string {
	out := make([]string, 0, len(mismatch.EntityIDs))
	for _, id := range mismatch.EntityIDs {
		out = append(out, names[id])
	}
	sort.Strings(out)
	return out
}

// sameStrings reports whether two string slices contain equal values in equal order.
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// ratio returns matched over expected, treating an empty expectation as complete recall.
func ratio(matched, expected int) float64 {
	if expected == 0 {
		return 1
	}
	return float64(matched) / float64(expected)
}

// precision returns matched over actual, treating no actual outputs as perfect precision.
func precision(matched, actual int) float64 {
	if actual == 0 {
		return 1
	}
	return float64(matched) / float64(actual)
}

// falsePositiveRate returns the share of actual outputs that did not match the golden expectation.
func falsePositiveRate(matched, actual int) float64 {
	if actual == 0 {
		return 0
	}
	return math.Max(0, float64(actual-matched)/float64(actual))
}
