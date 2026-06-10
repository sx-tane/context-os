package workspace

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/repository"
)

// Root dispatches method-specific handlers for /workspace.
func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.List(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		h.Delete(w, r)
		return
	}
	response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or DELETE required")
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
	response.WriteJSON(w, http.StatusOK, response.NewWorkspaceList(ws))
}

// Upsert handles POST /workspace.
//
// @Summary      Register or update a workspace
// @Description  Creates or updates a workspace record by its path.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      request.WorkspaceUpsert  true  "Workspace upsert request"
// @Success      200   {object}  repository.Workspace
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace/upsert [post]
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.WorkspaceUpsert
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
	response.WriteJSON(w, http.StatusOK, response.NewWorkspace(ws))
}

// Reset handles POST /workspace/reset.
//
// @Summary      Reset workspace memory
// @Description  Deletes DB-backed local memory and workspace-scoped local JSON artifacts, then recreates an empty workspace row.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      request.WorkspaceUpsert  true  "Workspace reset request"
// @Success      200   {object}  response.WorkspaceStatus
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace/reset [post]
func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.WorkspaceUpsert
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "path is required")
		return
	}
	resetter, ok := h.workspaces.(repository.WorkspaceResetter)
	if !ok {
		response.WriteError(w, http.StatusServiceUnavailable, "reset_unavailable", "workspace reset is unavailable for this store")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
		name = parts[len(parts)-1]
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	workspace, err := h.workspaces.GetByPath(ctx, path)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if err := resetter.DeleteByPath(ctx, path); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if err := h.deleteLocalWorkspaceArtifacts(workspaceIDForCleanup(path, workspace)); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "local_cleanup_error", err.Error())
		return
	}
	ws, err := h.workspaces.Upsert(ctx, repository.Workspace{Name: name, Path: path})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, response.NewWorkspaceStatus(ws, 0, 0, 0, 0, 0, 0, nil))
}

// Delete handles DELETE /workspace?path=<path>.
//
// @Summary      Delete workspace memory
// @Description  Deletes a workspace row, cascade-linked DB memory, and workspace-scoped local JSON artifacts without recreating the workspace.
// @Tags         workspace
// @Produce      json
// @Param        path  query     string  true  "Absolute workspace path"
// @Success      200   {object}  map[string]bool
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "DELETE required")
		return
	}

	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "path query parameter is required")
		return
	}
	resetter, ok := h.workspaces.(repository.WorkspaceResetter)
	if !ok {
		response.WriteError(w, http.StatusServiceUnavailable, "delete_unavailable", "workspace delete is unavailable for this store")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	workspace, err := h.workspaces.GetByPath(ctx, path)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if err := resetter.DeleteByPath(ctx, path); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if err := h.deleteLocalWorkspaceArtifacts(workspaceIDForCleanup(path, workspace)); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "local_cleanup_error", err.Error())
		return
	}
	workspace, err = h.workspaces.GetByPath(ctx, path)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	if workspace != nil {
		response.WriteError(w, http.StatusInternalServerError, "delete_incomplete", "workspace row still exists after delete")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
