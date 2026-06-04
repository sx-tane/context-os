package relationship_test

import (
	"context"
	"errors"
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/relationship"
)

// canonical wraps an entity for terse test setup.
func canonical(id, source string, kind types.EntityType, name string) entities.CanonicalEntity {
	return entities.CanonicalEntity{Entity: types.Entity{ID: id, SourceID: source, Type: kind, Name: name}}
}

// findKind returns the first relationship of the given kind, or false when none exists.
func findKind(rels []types.Relationship, kind types.RelationshipKind) (types.Relationship, bool) {
	for _, rel := range rels {
		if rel.Kind == kind {
			return rel, true
		}
	}
	return types.Relationship{}, false
}

type fakeAssistant struct {
	rels     []types.Relationship
	err      error
	provider string
	calls    int
}

// ProposeRelationships returns preconfigured test relationships.
func (f *fakeAssistant) ProposeRelationships(context.Context, types.NormalizedDocument, []entities.CanonicalEntity) ([]types.Relationship, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]types.Relationship, len(f.rels))
	copy(out, f.rels)
	return out, nil
}

// Provider returns the test provider label.
func (f *fakeAssistant) Provider() string {
	if f.provider != "" {
		return f.provider
	}
	return "fake"
}

// assistDoc returns a normalized document with relationship evidence text.
func assistDoc() types.NormalizedDocument {
	return types.NormalizedDocument{
		ID:          "d",
		Title:       "Checkout graph",
		Body:        "Checkout fee rule requires checkoutFeeAmount. checkoutFeeAmount is stored in checkout_fee_amount.",
		ContentHash: "doc-hash",
	}
}

// TestBuildOnlyLinksEntitiesFromSameSource verifies edges never cross document boundaries.
func TestBuildOnlyLinksEntitiesFromSameSource(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("doc-1:a", "doc-1", types.Requirement, "checkout requirement"),
		canonical("doc-2:b", "doc-2", types.APIField, "amount"),
	}

	got := relationship.Build(input)

	if len(got) != 0 {
		t.Fatalf("Build() length = %d, want 0", len(got))
	}
}

// TestBuildEmitsTypedDeliveryEdges verifies entity-type pairs map to the typed vocabulary.
func TestBuildEmitsTypedDeliveryEdges(t *testing.T) {
	tests := []struct {
		name     string
		from     entities.CanonicalEntity
		to       entities.CanonicalEntity
		wantKind types.RelationshipKind
		wantFrom string
		wantTo   string
	}{
		{
			name:     "requirement affects api",
			from:     canonical("d:req", "d", types.Requirement, "checkout"),
			to:       canonical("d:api", "d", types.APIField, "amount"),
			wantKind: types.RequirementAffectsAPI,
			wantFrom: "d:req",
			wantTo:   "d:api",
		},
		{
			name:     "api backed by db",
			from:     canonical("d:api", "d", types.APIField, "amount"),
			to:       canonical("d:col", "d", types.DBColumn, "amount_cents"),
			wantKind: types.APIBackedByDB,
			wantFrom: "d:api",
			wantTo:   "d:col",
		},
		{
			name:     "service depends on dependency",
			from:     canonical("d:svc", "d", types.Service, "billing"),
			to:       canonical("d:dep", "d", types.Dependency, "stripe"),
			wantKind: types.ServiceDependsOn,
			wantFrom: "d:svc",
			wantTo:   "d:dep",
		},
		{
			name:     "reversed input is oriented consistently",
			from:     canonical("d:api", "d", types.APIField, "amount"),
			to:       canonical("d:req", "d", types.Requirement, "checkout"),
			wantKind: types.RequirementAffectsAPI,
			wantFrom: "d:req",
			wantTo:   "d:api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relationship.Build([]entities.CanonicalEntity{tt.from, tt.to})
			rel, ok := findKind(got, tt.wantKind)
			if !ok {
				t.Fatalf("Build() missing kind %q, got %+v", tt.wantKind, got)
			}
			if rel.FromID != tt.wantFrom {
				t.Errorf("FromID = %q, want %q", rel.FromID, tt.wantFrom)
			}
			if rel.ToID != tt.wantTo {
				t.Errorf("ToID = %q, want %q", rel.ToID, tt.wantTo)
			}
			if rel.Confidence <= 0 {
				t.Errorf("Confidence = %v, want > 0", rel.Confidence)
			}
			if len(rel.Evidence) == 0 {
				t.Errorf("Evidence = empty, want source references")
			}
			if rel.Metadata["source_id"] != "d" {
				t.Errorf("Metadata[source_id] = %q, want d", rel.Metadata["source_id"])
			}
		})
	}
}

// TestBuildFallsBackToCoOccurrence verifies untyped pairs still produce an auditable edge.
func TestBuildFallsBackToCoOccurrence(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("d:dep1", "d", types.Dependency, "kafka"),
		canonical("d:dep2", "d", types.Dependency, "redis"),
	}

	got := relationship.Build(input)

	rel, ok := findKind(got, types.CoOccursInDocument)
	if !ok {
		t.Fatalf("Build() missing co-occurrence edge, got %+v", got)
	}
	if rel.Confidence != 0.5 {
		t.Errorf("Confidence = %v, want 0.5", rel.Confidence)
	}
}

// TestBuildSkipsLowConfidenceRegexOnlyCoOccurrence verifies generic regex-only dependency pairs do not create noisy edges.
func TestBuildSkipsLowConfidenceRegexOnlyCoOccurrence(t *testing.T) {
	input := []entities.CanonicalEntity{
		{Entity: types.Entity{ID: "d:dep1", SourceID: "d", Type: types.Dependency, Name: "kafka", ExtractionMethod: "regex_token", Confidence: 0.58}},
		{Entity: types.Entity{ID: "d:dep2", SourceID: "d", Type: types.Dependency, Name: "redis", ExtractionMethod: "regex_token", Confidence: 0.58}},
	}

	got := relationship.Build(input)

	if _, ok := findKind(got, types.CoOccursInDocument); ok {
		t.Fatalf("Build() emitted low-confidence regex co-occurrence edge: %+v", got)
	}
}

// TestBuildCapsGenericCoOccurrenceEdges verifies large same-document inputs do
// not explode into dense generic all-pairs graphs.
func TestBuildCapsGenericCoOccurrenceEdges(t *testing.T) {
	input := make([]entities.CanonicalEntity, 0, 100)
	for i := 0; i < 100; i++ {
		input = append(input, canonical(
			"d:dep"+string(rune('a'+(i%26)))+string(rune('a'+(i/26))),
			"d",
			types.Dependency,
			"dependency",
		))
	}

	got := relationship.Build(input)

	var generic int
	for _, rel := range got {
		if rel.Kind == types.CoOccursInDocument {
			generic++
		}
	}
	if generic > 25 {
		t.Fatalf("co-occurrence edge count = %d, want <= 25", generic)
	}
}

// TestBuildDeterministicIDIncludesKind verifies distinct edge kinds never collide on ID.
func TestBuildDeterministicIDIncludesKind(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Requirement, "checkout"),
		canonical("d:api", "d", types.APIField, "amount"),
	}

	got := relationship.Build(input)

	rel, ok := findKind(got, types.RequirementAffectsAPI)
	if !ok {
		t.Fatalf("Build() missing typed edge, got %+v", got)
	}
	want := "d:req->d:api:" + string(types.RequirementAffectsAPI)
	if rel.ID != want {
		t.Errorf("ID = %q, want %q", rel.ID, want)
	}
}

// TestParseAssistantOutputRejectsUnknownEntityNames verifies proposals cannot invent endpoints.
func TestParseAssistantOutputRejectsUnknownEntityNames(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"missingEntity","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"checkoutFeeAmount is stored in checkout_fee_amount","confidence":0.9}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 0", len(got))
	}
}

// TestParseAssistantOutputRejectsUnknownRelationshipKinds verifies proposals must use the stable relationship vocabulary.
func TestParseAssistantOutputRejectsUnknownRelationshipKinds(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"made_up","evidence":"checkoutFeeAmount is stored in checkout_fee_amount","confidence":0.9}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 0", len(got))
	}
}

// TestParseAssistantOutputRejectsMissingEvidence verifies proposals need source evidence.
func TestParseAssistantOutputRejectsMissingEvidence(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"","confidence":0.9}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 0", len(got))
	}
}

// TestParseAssistantOutputRejectsEvidenceOutsideSource verifies evidence must quote the same document.
func TestParseAssistantOutputRejectsEvidenceOutsideSource(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"checkoutFeeAmount is persisted in an external warehouse","confidence":0.9}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 0", len(got))
	}
}

// TestParseAssistantOutputRejectsLowConfidence verifies weak proposals stay below the acceptance gate.
func TestParseAssistantOutputRejectsLowConfidence(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"checkoutFeeAmount is stored in checkout_fee_amount","confidence":0.74}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 0", len(got))
	}
}

// TestParseAssistantOutputAcceptsEvidenceBackedEdge verifies valid proposals are converted into typed relationships.
func TestParseAssistantOutputAcceptsEvidenceBackedEdge(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
		canonical("d:db", "d", types.DBColumn, "checkout_fee_amount"),
	}
	output := `CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"checkoutFeeAmount is stored in checkout_fee_amount","confidence":0.86}]}`

	got, err := relationship.ParseAssistantOutput(output, doc, input)
	if err != nil {
		t.Fatalf("ParseAssistantOutput() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ParseAssistantOutput() length = %d, want 1", len(got))
	}
	rel := got[0]
	if rel.ID != "d:api->d:db:api_backed_by_db" {
		t.Fatalf("ID = %q, want deterministic relationship ID", rel.ID)
	}
	if rel.Metadata[relationship.MetadataAssistive] != "true" {
		t.Fatalf("Metadata[assistive] = %q, want true", rel.Metadata[relationship.MetadataAssistive])
	}
	if rel.Metadata[relationship.MetadataAssistProvider] != relationship.AssistProviderCodexCLI {
		t.Fatalf("Metadata[assist_provider] = %q, want codex_cli", rel.Metadata[relationship.MetadataAssistProvider])
	}
}

// TestBuildWithAssistDedupeAgainstDeterministicEdges verifies assist proposals cannot duplicate baseline edges.
func TestBuildWithAssistDedupeAgainstDeterministicEdges(t *testing.T) {
	doc := types.NormalizedDocument{ID: "d", Body: "checkout requirement drives amount"}
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Requirement, "checkout requirement"),
		canonical("d:api", "d", types.APIField, "amount"),
	}
	assistant := &fakeAssistant{rels: []types.Relationship{{
		FromID:     "d:req",
		ToID:       "d:api",
		Kind:       types.RequirementAffectsAPI,
		Confidence: 0.91,
		Evidence:   []string{"checkout requirement drives amount"},
	}}}

	got := relationship.BuildWithAssist(context.Background(), doc, input, assistant)

	if len(got) != len(relationship.Build(input)) {
		t.Fatalf("BuildWithAssist() length = %d, want deterministic length", len(got))
	}
}

// TestBuildWithAssistPopulatesAssistiveMetadata verifies accepted edges record provider and evidence metadata.
func TestBuildWithAssistPopulatesAssistiveMetadata(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Dependency, "Checkout fee rule"),
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
	}
	assistant := &fakeAssistant{provider: "fake_provider", rels: []types.Relationship{{
		FromID:     "d:req",
		ToID:       "d:api",
		Kind:       types.RequirementAffectsAPI,
		Confidence: 0.88,
		Evidence:   []string{"Checkout fee rule requires checkoutFeeAmount"},
	}}}

	got := relationship.BuildWithAssist(context.Background(), doc, input, assistant)

	rel, ok := findKind(got, types.RequirementAffectsAPI)
	if !ok {
		t.Fatalf("BuildWithAssist() missing assistive relationship, got %+v", got)
	}
	if rel.Metadata[relationship.MetadataAssistive] != "true" {
		t.Fatalf("Metadata[assistive] = %q, want true", rel.Metadata[relationship.MetadataAssistive])
	}
	if rel.Metadata[relationship.MetadataAssistProvider] != "fake_provider" {
		t.Fatalf("Metadata[assist_provider] = %q, want fake_provider", rel.Metadata[relationship.MetadataAssistProvider])
	}
	if rel.Metadata[relationship.MetadataAssistEvidence] != "Checkout fee rule requires checkoutFeeAmount" {
		t.Fatalf("Metadata[assist_evidence] = %q, want source quote", rel.Metadata[relationship.MetadataAssistEvidence])
	}
}

// TestBuildWithAssistFallsBackOnAssistantFailure verifies assistant errors preserve deterministic output.
func TestBuildWithAssistFallsBackOnAssistantFailure(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Requirement, "Checkout fee rule"),
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
	}
	assistant := &fakeAssistant{err: errors.New("codex unavailable")}

	got := relationship.BuildWithAssist(context.Background(), doc, input, assistant)
	want := relationship.Build(input)

	if len(got) != len(want) {
		t.Fatalf("BuildWithAssist() length = %d, want %d", len(got), len(want))
	}
	if assistant.calls != 1 {
		t.Fatalf("assistant calls = %d, want 1", assistant.calls)
	}
}

// TestCachedAssistantReusesDiskResult verifies relationship assistance is cached by document and entity IDs.
func TestCachedAssistantReusesDiskResult(t *testing.T) {
	doc := assistDoc()
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Dependency, "Checkout fee rule"),
		canonical("d:api", "d", types.APIField, "checkoutFeeAmount"),
	}
	fake := &fakeAssistant{rels: []types.Relationship{{
		FromID:     "d:req",
		ToID:       "d:api",
		Kind:       types.RequirementAffectsAPI,
		Confidence: 0.88,
		Evidence:   []string{"Checkout fee rule requires checkoutFeeAmount"},
	}}}
	cached := relationship.NewCachedAssistant(t.TempDir(), fake)

	first, err := cached.ProposeRelationships(context.Background(), doc, input)
	if err != nil {
		t.Fatalf("ProposeRelationships() error = %v", err)
	}
	second, err := cached.ProposeRelationships(context.Background(), doc, input)
	if err != nil {
		t.Fatalf("ProposeRelationships() cached error = %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("assistant calls = %d, want 1", fake.calls)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("cached relationship lengths = %d/%d, want 1/1", len(first), len(second))
	}
}

// TestValidateRejectsInvalidEdges verifies structurally invalid edges are reported.
func TestValidateRejectsInvalidEdges(t *testing.T) {
	tests := []struct {
		name string
		rel  types.Relationship
	}{
		{name: "empty from", rel: types.Relationship{ID: "x", ToID: "b", Kind: types.APIBackedByDB}},
		{name: "empty to", rel: types.Relationship{ID: "x", FromID: "a", Kind: types.APIBackedByDB}},
		{name: "self loop", rel: types.Relationship{ID: "x", FromID: "a", ToID: "a", Kind: types.APIBackedByDB}},
		{name: "empty kind", rel: types.Relationship{ID: "x", FromID: "a", ToID: "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := relationship.Validate(tt.rel); err == nil {
				t.Errorf("Validate() error = nil, want error")
			}
		})
	}

	valid := types.Relationship{ID: "x", FromID: "a", ToID: "b", Kind: types.APIBackedByDB}
	if err := relationship.Validate(valid); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}
