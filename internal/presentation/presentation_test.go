package presentation_test

import (
	"strings"
	"testing"

	"context-os/domain/types"
	"context-os/internal/presentation"
)

// TestRenderSummaryReturnsCleanState verifies a role summary reports no delivery mismatches when no findings exist.
func TestRenderSummaryReturnsCleanState(t *testing.T) {
	got := presentation.RenderSummary(presentation.PMO, nil)
	if got != "pmo view: no delivery mismatches detected" {
		t.Fatalf("RenderSummary() = %q, want clean state", got)
	}
}

// TestRenderSummaryIncludesConfidenceImpactAndEvidence verifies explainability fields surface in the rendered bullet.
func TestRenderSummaryIncludesConfidenceImpactAndEvidence(t *testing.T) {
	got := presentation.RenderSummary(presentation.PMO, []types.Mismatch{{
		Summary:    "Potential delivery mismatch around missingRefundState",
		Severity:   "medium",
		Confidence: 0.70,
		Impact:     "medium",
		Evidence:   []string{"repo://refund-flow#missingRefundState"},
	}})

	for _, want := range []string{"confidence 0.70", "impact medium", "evidence: repo://refund-flow#missingRefundState"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderSummary() = %q, want substring %q", got, want)
		}
	}
}

// TestRenderSummaryOmitsUnsetExplainabilityFields verifies findings without confidence/impact/evidence render only severity and summary.
func TestRenderSummaryOmitsUnsetExplainabilityFields(t *testing.T) {
	got := presentation.RenderSummary(presentation.PMO, []types.Mismatch{{
		Summary:  "Potential delivery mismatch around missingRefundState",
		Severity: "medium",
	}})

	if strings.Contains(got, "confidence") || strings.Contains(got, "impact") || strings.Contains(got, "evidence") {
		t.Fatalf("RenderSummary() = %q, want no explainability qualifiers", got)
	}
}
func TestRenderSummaryIncludesRoleSeverityAndSummary(t *testing.T) {
	got := presentation.RenderSummary(presentation.ServiceLayer, []types.Mismatch{{
		Summary:  "Potential delivery mismatch around missingRefundState",
		Severity: "medium",
	}})

	for _, want := range []string{"service_layer view: 1 delivery mismatch(es) detected", "[medium]", "missingRefundState"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderSummary() = %q, want substring %q", got, want)
		}
	}
}
