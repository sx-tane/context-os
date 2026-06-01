// Package workspace provides HTTP handlers for ContextOS workspace management.
package workspace

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

const workspaceRequestTimeout = 10 * time.Second

// Handler holds the repository dependencies for workspace HTTP handlers.
type Handler struct {
	workspaces  repository.WorkspaceRepository
	events      repository.EventRepository
	connSync    repository.SyncRepository
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	connSync repository.SyncRepository,
) *Handler {
	return &Handler{
		workspaces: workspaces,
		events:     events,
		connSync:   connSync,
	}
}

// List handles GET /workspace.
//
// @Summary      List workspaces
// @Description  Returns all registered ContextOS workspaces.
// @Tags         workspace
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /workspace [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	ws, err := h.workspaces.List(ctx)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"workspaces": ws,
		"count":      len(ws),
	})
}

// Upsert handles POST /workspace.
//
// @Summary      Register or update a workspace
// @Description  Creates or updates a workspace record by its path.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      upsertRequest  true  "Workspace upsert request"
// @Success      200   {object}  repository.Workspace
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace [post]
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req upsertRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "path is required")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		// Derive name from last path component.
		parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
		name = parts[len(parts)-1]
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	ws, err := h.workspaces.Upsert(ctx, repository.Workspace{
		Name: name,
		Path: path,
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, ws)
}

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

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"workspace":   ws,
		"event_count": eventCount,
		"syncs":       syncs,
	})
}

// upsertRequest is the decoded body for POST /workspace.
type upsertRequest struct {
	// Path is the absolute local folder path for the workspace.
	Path string `json:"path"`
	// Name is the optional human-readable workspace name.
	Name string `json:"name"`
}
