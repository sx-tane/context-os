package presentation

// White-box tests cover role summary helpers split out of presentation.go.

import (
	"strings"
	"testing"

	"context-os/domain/types"
	stagepresentation "context-os/internal/stages/presentation"
)

// TestParseRoleDefaultsToPMO verifies unknown role input falls back to the PMO view.
func TestParseRoleDefaultsToPMO(t *testing.T) {
	t.Parallel()

	if got := parseRole("unknown"); got != stagepresentation.PMO {
		t.Fatalf("parseRole() = %q, want %q", got, stagepresentation.PMO)
	}
}

// TestSplitPresentationFindingsSeparatesReviewCandidates verifies dependency risks are removed from actionable findings.
func TestSplitPresentationFindingsSeparatesReviewCandidates(t *testing.T) {
	t.Parallel()

	actionable, review := splitPresentationFindings([]types.Mismatch{
		{ID: "m1", Type: "contract_mismatch"},
		{ID: "dependency_risk:svc", Type: "dependency_risk"},
	})
	if len(actionable) != 1 {
		t.Fatalf("actionable count = %d, want 1", len(actionable))
	}
	if len(review) != 1 {
		t.Fatalf("review count = %d, want 1", len(review))
	}
	if review[0].Type != "dependency_review" {
		t.Fatalf("review type = %q, want dependency_review", review[0].Type)
	}
}

// TestRenderFindingsSummaryMentionsReviewCandidates verifies summaries disclose hidden dependency review candidates.
func TestRenderFindingsSummaryMentionsReviewCandidates(t *testing.T) {
	t.Parallel()

	summary := renderFindingsSummary(stagepresentation.PMO, nil, []types.Mismatch{{ID: "dependency_risk:svc"}})
	if !strings.Contains(summary, "Review candidates: 1") {
		t.Fatalf("summary = %q, want review candidate count", summary)
	}
}
