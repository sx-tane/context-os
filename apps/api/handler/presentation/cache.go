package presentation

import (
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/types"
	"net/http"
	"strings"
)

// writeFindingsFromCache returns persisted mismatches without re-ingesting.
func (h *Handler) writeFindingsFromCache(w http.ResponseWriter, req request.PresentationFindings, workspaceID string, cached []types.Mismatch) {
	actionable, reviewCandidates := splitPresentationFindings(cached)
	role := parseRole(req.Role)
	mismatchIDs := collectMismatchIDs(actionable)
	traceID := buildTraceID(req.Connector, req.URI, mismatchIDs)
	views := buildRoleViews(actionable)

	response.WriteJSON(w, http.StatusOK, response.PresentationFindings{
		Connector:            strings.ToLower(strings.TrimSpace(req.Connector)),
		URI:                  strings.TrimSpace(req.URI),
		Role:                 string(role),
		TraceID:              traceID,
		Summary:              renderFindingsSummary(role, actionable, reviewCandidates),
		MismatchCount:        len(actionable),
		ReviewCandidateCount: len(reviewCandidates),
		SeverityCount:        severityCount(actionable),
		MismatchIDs:          mismatchIDs,
		Mismatches:           actionable,
		ReviewCandidates:     reviewCandidates,
		Views:                views,
		PMO:                  buildPMOSummary(actionable),
		Execution: response.ExecutionEvidence{
			Enabled:   false,
			Assistive: true,
			Summary:   "results from cache",
			Metadata: map[string]string{
				"trace_id":     traceID,
				"workspace_id": workspaceID,
				"source":       "cache",
			},
		},
	})
}
