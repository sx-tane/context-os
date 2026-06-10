package workspace

import (
	"context"
	"net/http"
	"strings"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

// Status handles GET /workspace/status?path=<path>.
//
// @Summary      Workspace status
// @Description  Returns event counts and connector sync states for a workspace.
// @Tags         workspace
// @Produce      json
// @Param        path  query     string  true  "Absolute workspace path"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace/status [get]
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "path query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	ws, err := h.workspaces.GetByPath(ctx, path)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if ws == nil {
		response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	eventCount, err := h.events.Count(ctx, ws.ID, "")
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	syncs, err := h.connSync.ListByWorkspace(ctx, ws.ID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	// Detailed counts are best-effort — missing repositories return 0.
	var workspaceCount, entityCount, relationshipCount, mismatchCount, auditCount int
	if workspaces, wErr := h.workspaces.List(ctx); wErr == nil {
		workspaceCount = len(workspaces)
	}
	if h.entities != nil {
		entityList, eErr := h.entities.ListEntities(ctx, ws.ID, "")
		if eErr == nil {
			entityCount = len(entityList)
		}
		if counter, ok := h.entities.(repository.RelationshipCounter); ok {
			if count, cErr := counter.CountRelationships(ctx, ws.ID); cErr == nil {
				relationshipCount = count
			}
		}
	}
	if h.mismatches != nil {
		mismatchList, mErr := h.mismatches.ListByWorkspace(ctx, ws.ID, "", 0)
		if mErr == nil {
			mismatchCount = len(mismatchList)
		}
	}
	if counter, ok := h.audit.(repository.AuditCounter); ok {
		if count, cErr := counter.CountByWorkspace(ctx, ws.ID); cErr == nil {
			auditCount = count
		}
	}

	response.WriteJSON(w, http.StatusOK, response.NewWorkspaceStatus(
		*ws,
		eventCount,
		entityCount,
		relationshipCount,
		mismatchCount,
		workspaceCount,
		auditCount,
		syncs,
	))
}
