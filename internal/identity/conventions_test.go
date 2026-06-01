package identity_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/identity"
)

// TestConventionAliasesGeneratesCanonicalForms verifies a name expands to every
// standard naming-convention spelling in a deterministic order.
func TestConventionAliasesGeneratesCanonicalForms(t *testing.T) {
	got := identity.ConventionAliases("refundStatus")
	want := []string{"refund_status", "refund-status", "refundStatus", "RefundStatus", "REFUND_STATUS"}
	if len(got) != len(want) {
		t.Fatalf("ConventionAliases() length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, form := range want {
		if got[i] != form {
			t.Errorf("ConventionAliases()[%d] = %q, want %q", i, got[i], form)
		}
	}
}

// TestConventionAliasesEmptyForBlankName verifies names with no word tokens yield no aliases.
func TestConventionAliasesEmptyForBlankName(t *testing.T) {
	if got := identity.ConventionAliases("   "); got != nil {
		t.Errorf("ConventionAliases(blank) = %v, want nil", got)
	}
}

// TestResolveMarksConventionMatchLayer verifies aliases linked only by spelling
// convention report the convention match layer.
func TestResolveMarksConventionMatchLayer(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:refund_status", Name: "refund_status", Type: types.APIField, SourceID: "doc-1"},
		{ID: "doc-1:refundStatus", Name: "refundStatus", Type: types.APIField, SourceID: "doc-1"},
	}

	got := identity.Resolve(input)
	if len(got) != 1 {
		t.Fatalf("Resolve() length = %d, want 1", len(got))
	}
	if got[0].MatchLayer != entities.MatchLayerConvention {
		t.Errorf("MatchLayer = %q, want %q", got[0].MatchLayer, entities.MatchLayerConvention)
	}
	if got[0].NeedsHuman {
		t.Errorf("NeedsHuman = true, want false for clean convention merge")
	}
}

// TestResolveFlagsTypeConflict verifies merged aliases that disagree on entity type
// are flagged for human review with a conflict reason and lowered confidence.
func TestResolveFlagsTypeConflict(t *testing.T) {
	input := []types.Entity{
		{ID: "doc-1:status", Name: "status", Type: types.APIField, SourceID: "doc-1"},
		{ID: "doc-2:status", Name: "status", Type: types.DBColumn, SourceID: "doc-2"},
	}

	got := identity.Resolve(input)
	if len(got) != 1 {
		t.Fatalf("Resolve() length = %d, want 1", len(got))
	}
	if !got[0].NeedsHuman {
		t.Errorf("NeedsHuman = false, want true on type conflict")
	}
	if got[0].Confidence >= 1 {
		t.Errorf("Confidence = %v, want < 1 on type conflict", got[0].Confidence)
	}
	if got[0].ConflictReason == "" {
		t.Errorf("ConflictReason = empty, want a populated reason")
	}
}

// TestResolveExactMatchPopulatesEvidence verifies a first-seen entity records exact-layer provenance.
func TestResolveExactMatchPopulatesEvidence(t *testing.T) {
	got := identity.Resolve([]types.Entity{
		{ID: "doc-1:refund", Name: "refund", Type: types.APIField, SourceID: "doc-1"},
	})
	if len(got) != 1 {
		t.Fatalf("Resolve() length = %d, want 1", len(got))
	}
	if got[0].MatchLayer != entities.MatchLayerExact {
		t.Errorf("MatchLayer = %q, want %q", got[0].MatchLayer, entities.MatchLayerExact)
	}
	if len(got[0].Evidence) == 0 {
		t.Errorf("Evidence = empty, want at least one provenance note")
	}
}
