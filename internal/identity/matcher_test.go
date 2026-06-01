package identity_test

import (
	"errors"
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/identity"
)

// stubMatcher returns preset similarity scores for named pairs, enabling
// deterministic semantic-layer tests independent of any scoring heuristic.
type stubMatcher struct {
	scores map[[2]string]float64
}

// Similarity looks up a and b in either order, defaulting to 0 when absent.
func (s stubMatcher) Similarity(a, b string) float64 {
	if v, ok := s.scores[[2]string{a, b}]; ok {
		return v
	}
	if v, ok := s.scores[[2]string{b, a}]; ok {
		return v
	}
	return 0
}

// failingEmbedder always returns an error, exercising fail-closed behaviour.
type failingEmbedder struct{}

// Embed always fails so callers can verify WorkerMatcher degrades safely.
func (failingEmbedder) Embed(texts []string) ([][]float64, error) {
	return nil, errors.New("worker unavailable")
}

// fixedEmbedder returns preset vectors per text for deterministic worker tests.
type fixedEmbedder struct {
	vectors map[string][]float64
}

// Embed returns the configured vector for each text, or a zero vector when absent.
func (f fixedEmbedder) Embed(texts []string) ([][]float64, error) {
	out := make([][]float64, len(texts))
	for i, text := range texts {
		out[i] = f.vectors[text]
	}
	return out, nil
}

// TestLocalMatcherSimilarityIsDeterministic verifies repeated scoring of the same
// inputs yields identical results and that identical strings score 1.
func TestLocalMatcherSimilarityIsDeterministic(t *testing.T) {
	m := identity.LocalMatcher{}
	first := m.Similarity("refund status", "refund state")
	second := m.Similarity("refund status", "refund state")
	if first != second {
		t.Errorf("Similarity() not deterministic: %v != %v", first, second)
	}
	if self := m.Similarity("refund", "refund"); self < 0.999 {
		t.Errorf("Similarity(identical) = %v, want ~1", self)
	}
}

// TestResolveWithMatcherAddsSemanticCandidates verifies a strong semantic pair is
// recorded as a review candidate on both entities without merging them.
func TestResolveWithMatcherAddsSemanticCandidates(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:refund_status", Name: "refund_status", Type: types.APIField, SourceID: "doc-1"},
		{ID: "doc-2:reimbursement_state", Name: "reimbursement_state", Type: types.APIField, SourceID: "doc-2"},
	}
	matcher := stubMatcher{scores: map[[2]string]float64{
		{"refund_status", "reimbursement_state"}: 0.9,
	}}

	got := identity.ResolveWithMatcher(input, matcher, identity.MatchOptions{})
	if len(got) != 2 {
		t.Fatalf("ResolveWithMatcher() length = %d, want 2 (no auto-merge)", len(got))
	}
	for _, ent := range got {
		if len(ent.Candidates) != 1 {
			t.Fatalf("entity %q candidates = %d, want 1", ent.Entity.Name, len(ent.Candidates))
		}
		if ent.Candidates[0].Layer != entities.MatchLayerSemantic {
			t.Errorf("candidate layer = %q, want %q", ent.Candidates[0].Layer, entities.MatchLayerSemantic)
		}
		if ent.Candidates[0].Accepted {
			t.Errorf("candidate accepted = true, want false (never auto-merge)")
		}
		if !ent.NeedsHuman {
			t.Errorf("entity %q NeedsHuman = false, want true", ent.Entity.Name)
		}
	}
}

// TestResolveWithMatcherIgnoresWeakPairs verifies pairs below threshold produce no candidates.
func TestResolveWithMatcherIgnoresWeakPairs(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:refund", Name: "refund", Type: types.APIField, SourceID: "doc-1"},
		{ID: "doc-2:invoice", Name: "invoice", Type: types.APIField, SourceID: "doc-2"},
	}
	matcher := stubMatcher{scores: map[[2]string]float64{
		{"refund", "invoice"}: 0.10,
	}}

	got := identity.ResolveWithMatcher(input, matcher, identity.MatchOptions{})
	for _, ent := range got {
		if len(ent.Candidates) != 0 {
			t.Errorf("entity %q candidates = %d, want 0", ent.Entity.Name, len(ent.Candidates))
		}
		if ent.NeedsHuman {
			t.Errorf("entity %q NeedsHuman = true, want false", ent.Entity.Name)
		}
	}
}

// TestResolveWithMatcherNilDefaultsToLocal verifies a nil matcher falls back to the
// deterministic local matcher and still resolves entities.
func TestResolveWithMatcherNilDefaultsToLocal(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:refund", Name: "refund", Type: types.APIField, SourceID: "doc-1"},
	}
	got := identity.ResolveWithMatcher(input, nil, identity.MatchOptions{})
	if len(got) != 1 {
		t.Fatalf("ResolveWithMatcher(nil) length = %d, want 1", len(got))
	}
}

// TestWorkerMatcherFailsClosed verifies an embedding error yields a zero score
// rather than panicking the pipeline.
func TestWorkerMatcherFailsClosed(t *testing.T) {
	m := identity.WorkerMatcher{Embedder: failingEmbedder{}}
	if score := m.Similarity("a", "b"); score != 0 {
		t.Errorf("Similarity() on error = %v, want 0", score)
	}
}

// TestWorkerMatcherScoresEmbeddings verifies identical embeddings score 1 and
// orthogonal embeddings score 0.
func TestWorkerMatcherScoresEmbeddings(t *testing.T) {
	m := identity.WorkerMatcher{Embedder: fixedEmbedder{vectors: map[string][]float64{
		"same_a":  {1, 0, 0},
		"same_b":  {1, 0, 0},
		"ortho_a": {1, 0, 0},
		"ortho_b": {0, 1, 0},
	}}}
	if score := m.Similarity("same_a", "same_b"); score < 0.999 {
		t.Errorf("Similarity(identical vectors) = %v, want ~1", score)
	}
	if score := m.Similarity("ortho_a", "ortho_b"); score != 0 {
		t.Errorf("Similarity(orthogonal vectors) = %v, want 0", score)
	}
}
