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

// TestRenderSummaryIncludesRoleSeverityAndSummary verifies mismatch summaries preserve role, count, severity, and finding text.
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
