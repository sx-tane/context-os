package identity_test

import (
	"encoding/csv"
	"os"
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/stages/identity"
)

// benchmarkDatasetPath locates the labelled identity dataset relative to this package.
const benchmarkDatasetPath = "../../../tests/harness/fixtures/identity/benchmark-dataset.csv"

// datasetRow is one labelled alias from the benchmark dataset.
type datasetRow struct {
	alias     string // surface form observed in a source
	canonical string // expected canonical identity this alias belongs to
	source    string // originating source label, used as the entity SourceID
}

// oracleMatcher simulates a perfect semantic embedder for the benchmark dataset:
// two names score high when the dataset labels them as the same canonical identity
// but exact and convention matching could not merge them. It isolates resolver and
// metric wiring from embedder quality, which is benchmarked separately once a real
// worker is connected.
type oracleMatcher struct {
	canonicalByName map[string]string
}

// Similarity returns 0.9 for distinct names sharing a canonical label, else 0.
func (o oracleMatcher) Similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	if o.canonicalByName[a] != "" && o.canonicalByName[a] == o.canonicalByName[b] {
		return 0.9
	}
	return 0
}

// TestBenchmarkIdentityResolution measures precision, recall, false-merge, conflict,
// and unresolved rates of the identity resolver against a labelled dataset covering
// exact, convention, multilingual, and semantic aliases plus negative pairs.
//
// Run: go test ./internal/stages/identity/... -run BenchmarkIdentityResolution -v
func TestBenchmarkIdentityResolution(t *testing.T) {
	rows := loadBenchmarkDataset(t)

	input := make([]types.Entity, 0, len(rows))
	canonicalByName := map[string]string{}
	for _, row := range rows {
		input = append(input, types.Entity{
			ID:       row.source + ":" + row.alias,
			Name:     row.alias,
			Type:     types.APIField,
			SourceID: row.source,
		})
		canonicalByName[row.alias] = row.canonical
	}

	resolved := identity.ResolveWithMatcher(input, oracleMatcher{canonicalByName: canonicalByName}, identity.MatchOptions{})

	// Map each canonical entity (by its primary name) to its true canonical label and
	// the set of names it is linked to via merge aliases or semantic candidates.
	type linkSet struct {
		canonical string
		linked    map[string]bool
		conflict  bool
		needhuman bool
	}
	resolvedSets := make([]linkSet, 0, len(resolved))
	for _, ent := range resolved {
		linked := map[string]bool{}
		for _, alias := range ent.Entity.Aliases {
			linked[alias] = true
		}
		for _, cand := range ent.Candidates {
			linked[cand.Alias] = true
		}
		resolvedSets = append(resolvedSets, linkSet{
			canonical: canonicalByName[ent.Entity.Name],
			linked:    linked,
			conflict:  ent.ConflictReason != "" && !hasSemanticCandidate(ent.Candidates),
			needhuman: ent.NeedsHuman,
		})
	}

	var truePos, falsePos, falseNeg, needHuman, conflicts int
	for i := range resolvedSets {
		if resolvedSets[i].needhuman {
			needHuman++
		}
		if resolvedSets[i].conflict {
			conflicts++
		}
		for j := i + 1; j < len(resolvedSets); j++ {
			predictedLink := sharesLinkedName(resolvedSets[i].linked, resolvedSets[j].linked)
			trueLink := resolvedSets[i].canonical == resolvedSets[j].canonical
			switch {
			case predictedLink && trueLink:
				truePos++
			case predictedLink && !trueLink:
				falsePos++
			case !predictedLink && trueLink:
				falseNeg++
			}
		}
	}

	precision := ratio(truePos, truePos+falsePos)
	recall := ratio(truePos, truePos+falseNeg)
	falsePositiveRate := ratio(falsePos, truePos+falsePos)

	t.Logf("identity benchmark: tp=%d fp=%d fn=%d precision=%.3f recall=%.3f fp_rate=%.3f needs_human=%d conflicts=%d",
		truePos, falsePos, falseNeg, precision, recall, falsePositiveRate, needHuman, conflicts)

	if precision < 0.99 {
		t.Errorf("precision = %.3f, want >= 0.99", precision)
	}
	if recall < 0.90 {
		t.Errorf("recall = %.3f, want >= 0.90", recall)
	}
	if falsePos != 0 {
		t.Errorf("false merges = %d, want 0", falsePos)
	}
	if conflicts != 0 {
		t.Errorf("type conflicts = %d, want 0 for this dataset", conflicts)
	}
}

// hasSemanticCandidate reports whether any candidate came from the semantic layer.
func hasSemanticCandidate(candidates []entities.MergeCandidate) bool {
	for _, c := range candidates {
		if c.Layer == entities.MatchLayerSemantic {
			return true
		}
	}
	return false
}

// sharesLinkedName reports whether two link sets reference any common name.
func sharesLinkedName(a, b map[string]bool) bool {
	for name := range a {
		if b[name] {
			return true
		}
	}
	return false
}

// ratio returns numerator/denominator, or 1.0 when the denominator is zero so an
// empty category does not falsely fail a threshold check.
func ratio(num, den int) float64 {
	if den == 0 {
		return 1
	}
	return float64(num) / float64(den)
}

// loadBenchmarkDataset reads and parses the labelled identity dataset CSV.
func loadBenchmarkDataset(t *testing.T) []datasetRow {
	t.Helper()
	file, err := os.Open(benchmarkDatasetPath)
	if err != nil {
		t.Fatalf("open dataset = %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read dataset = %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("dataset rows = %d, want header plus data", len(records))
	}

	rows := make([]datasetRow, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) < 6 {
			t.Fatalf("dataset record columns = %d, want 6", len(record))
		}
		rows = append(rows, datasetRow{alias: record[0], canonical: record[1], source: record[3]})
	}
	return rows
}
