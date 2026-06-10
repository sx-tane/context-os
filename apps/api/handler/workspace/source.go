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

// Source handles POST /workspace/source.
//
// @Summary      Register a connected workspace source
// @Description  Saves a connector/source URI reference for live source lookup without ingesting source content.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      request.WorkspaceSource  true  "Connected source registration request"
// @Success      200   {object}  response.WorkspaceSync
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /workspace/source [post]
func (h *Handler) Source(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	if h.connSync == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "sync_unavailable", "workspace source registration is unavailable for this store")
		return
	}

	var req request.WorkspaceSource
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	workspaceID := strings.TrimSpace(req.WorkspaceID)
	connector := strings.ToLower(strings.TrimSpace(req.Connector))
	sourceURI := strings.TrimSpace(firstNonEmpty(req.SourceURI, req.URI))
	if workspaceID == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id is required")
		return
	}
	if connector == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "connector is required")
		return
	}
	if sourceURI == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "source_uri is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	workspace, err := h.resolveOrCreateWorkspace(ctx, workspaceID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	sync := repository.ConnectorSync{
		WorkspaceID: workspace.ID,
		Connector:   connector,
		SourceURI:   sourceURI,
		EventCount:  0,
		Status:      "connected",
	}
	if err := h.connSync.Upsert(ctx, sync); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, response.NewWorkspaceSync(sync))
}

// resolveOrCreateWorkspace finds an existing workspace by path or ID, or creates one from the reference.
func (h *Handler) resolveOrCreateWorkspace(ctx context.Context, ref string) (repository.Workspace, error) {
	workspace, err := h.workspaces.GetByPath(ctx, ref)
	if err != nil {
		return repository.Workspace{}, err
	}
	if workspace != nil {
		return *workspace, nil
	}

	workspaces, err := h.workspaces.List(ctx)
	if err != nil {
		return repository.Workspace{}, err
	}
	for _, workspace := range workspaces {
		if workspace.ID == ref {
			return workspace, nil
		}
	}

	name := workspaceNameFromPath(ref)
	return h.workspaces.Upsert(ctx, repository.Workspace{
		Name: name,
		Path: ref,
	})
}

// workspaceNameFromPath derives a human-readable workspace name from a path-like reference.
func workspaceNameFromPath(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return "workspace"
	}
	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return "workspace"
	}
	return name
}
