package workspace

// White-box tests cover UI state payload validation helpers split out of workspace.go.

import (
	"encoding/json"
	"testing"

	"context-os/apps/api/request"
)

// TestValidateAnalysisBasketPayloadNormalizesNilItems verifies missing basket items persist as an empty JSON array.
func TestValidateAnalysisBasketPayloadNormalizesNilItems(t *testing.T) {
	t.Parallel()

	workspaceID, normalized, err := validateAnalysisBasketPayload([]byte(`{"workspace_id":"ws1"}`))
	if err != nil {
		t.Fatalf("validateAnalysisBasketPayload() error = %v", err)
	}
	if workspaceID != "ws1" {
		t.Fatalf("workspaceID = %q, want ws1", workspaceID)
	}
	var payload request.AnalysisBasket
	if err := json.Unmarshal(normalized, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Items == nil {
		t.Fatal("Items = nil, want empty slice")
	}
}

// TestValidateFindingActionsPayloadRejectsUnsupportedStatus verifies invalid checklist statuses fail validation.
func TestValidateFindingActionsPayloadRejectsUnsupportedStatus(t *testing.T) {
	t.Parallel()

	_, _, err := validateFindingActionsPayload([]byte(`{"workspace_id":"ws1","actions":[{"findingId":"m1","status":"blocked"}]}`))
	if err == nil {
		t.Fatal("validateFindingActionsPayload() error = nil, want invalid status error")
	}
}
