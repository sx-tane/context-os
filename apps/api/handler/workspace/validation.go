package workspace

import (
	"encoding/json"
	"fmt"
	"strings"

	"context-os/apps/api/request"
)

// validateAnalysisBasketPayload validates and normalizes analysis basket JSON before persistence.
func validateAnalysisBasketPayload(body []byte) (string, []byte, error) {
	var payload request.AnalysisBasket
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	workspaceID := strings.TrimSpace(payload.WorkspaceID)
	if workspaceID == "" {
		return "", nil, fmt.Errorf("workspace_id is required")
	}
	if payload.Items == nil {
		payload.Items = []request.AnalysisBasketItem{}
	}
	for index, item := range payload.Items {
		if strings.TrimSpace(item.ID) == "" {
			return "", nil, fmt.Errorf("items[%d].id is required", index)
		}
		if strings.TrimSpace(item.Connector) == "" {
			return "", nil, fmt.Errorf("items[%d].connector is required", index)
		}
		if strings.TrimSpace(item.URI) == "" {
			return "", nil, fmt.Errorf("items[%d].uri is required", index)
		}
		if strings.TrimSpace(item.Label) == "" {
			return "", nil, fmt.Errorf("items[%d].label is required", index)
		}
		if strings.TrimSpace(item.Origin) == "" {
			return "", nil, fmt.Errorf("items[%d].origin is required", index)
		}
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	return workspaceID, normalized, nil
}

// validateFindingActionsPayload validates and normalizes finding action checklist JSON before persistence.
func validateFindingActionsPayload(body []byte) (string, []byte, error) {
	var payload request.FindingActions
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	workspaceID := strings.TrimSpace(payload.WorkspaceID)
	if workspaceID == "" {
		return "", nil, fmt.Errorf("workspace_id is required")
	}
	if payload.Actions == nil {
		payload.Actions = []request.FindingActionItem{}
	}
	for index, item := range payload.Actions {
		if strings.TrimSpace(item.FindingID) == "" {
			return "", nil, fmt.Errorf("actions[%d].findingId is required", index)
		}
		switch strings.TrimSpace(item.Status) {
		case "open", "checking", "done", "ignored", "false_positive":
		default:
			return "", nil, fmt.Errorf("actions[%d].status must be open, checking, done, ignored, or false_positive", index)
		}
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	return workspaceID, normalized, nil
}

// emptyAnalysisBasket returns the default empty analysis basket payload for a workspace.
func emptyAnalysisBasket(workspaceID string) any {
	return request.AnalysisBasket{WorkspaceID: workspaceID, Items: []request.AnalysisBasketItem{}}
}

// emptyFindingActions returns the default empty finding action checklist payload for a workspace.
func emptyFindingActions(workspaceID string) any {
	return request.FindingActions{WorkspaceID: workspaceID, Actions: []request.FindingActionItem{}}
}
