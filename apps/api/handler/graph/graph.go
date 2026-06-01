// Package graph provides the HTTP handler for querying persisted entity graph data.
package graph

import (
	"net/http"
	"strings"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

// Handler exposes graph query endpoints backed by persistent entity data.
type Handler struct {
	workspaces repository.WorkspaceRepository
	entities   repository.EntityRepository
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(workspaces repository.WorkspaceRepository, entities repository.EntityRepository) *Handler {
	return &Handler{workspaces: workspaces, entities: entities}
}

// Query handles GET /graph.
//
// @Summary      Query workspace entity graph
// @Description  Returns persisted canonical entities for a workspace, optionally filtered by entity type.
// @Tags         graph
// @Produce      json
// @Param        workspace_id  query     string  true   "Workspace path or ID"
// @Param        entity_type   query     string  false  "Filter by entity type (e.g. feature, person, service)"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /graph [get]
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if workspaceID == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id is required")
		return
	}
	entityType := strings.TrimSpace(r.URL.Query().Get("entity_type"))

	// Resolve workspace path to stored ID via WorkspaceRepository.
	ws, err := h.workspaces.GetByPath(r.Context(), workspaceID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	resolvedID := workspaceID
	if ws != nil {
		resolvedID = ws.ID
	}

	canonical, err := h.entities.ListEntities(r.Context(), resolvedID, entityType)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"workspace_id":  resolvedID,
		"entity_type":   entityType,
		"entity_count":  len(canonical),
		"entities":      canonical,
	})
}
