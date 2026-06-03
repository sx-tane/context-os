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
	workspaces repository.WorkspaceRepository
	events     repository.EventRepository
	entities   repository.EntityRepository
	mismatches repository.MismatchRepository
	connSync   repository.SyncRepository
	audit      repository.AuditRepository
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	entities repository.EntityRepository,
	mismatches repository.MismatchRepository,
	connSync repository.SyncRepository,
) *Handler {
	return &Handler{
		workspaces: workspaces,
		events:     events,
		entities:   entities,
		mismatches: mismatches,
		connSync:   connSync,
	}
}

// WithAuditRepository attaches audit row counts to workspace status.
func (h *Handler) WithAuditRepository(audit repository.AuditRepository) *Handler {
	h.audit = audit
	return h
}

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
// @Param        body  body      upsertRequest  true  "Workspace upsert request"
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
	response.WriteJSON(w, http.StatusOK, response.NewWorkspace(ws))
}

// Reset handles POST /workspace/reset.
//
// @Summary      Reset workspace memory
// @Description  Deletes DB-backed local memory for a workspace and recreates an empty workspace row.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      upsertRequest  true  "Workspace reset request"
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

	if err := resetter.DeleteByPath(ctx, path); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
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
// @Description  Deletes a workspace row and all cascade-linked local memory without recreating the workspace.
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

	if err := resetter.DeleteByPath(ctx, path); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	workspace, err := h.workspaces.GetByPath(ctx, path)
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

// Source handles POST /workspace/source.
//
// @Summary      Register a connected workspace source
// @Description  Saves a connector/source URI reference for live source lookup without ingesting source content.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        body  body      sourceRequest  true  "Connected source registration request"
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

	var req sourceRequest
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

// upsertRequest is the decoded body for POST /workspace.
type upsertRequest struct {
	// Path is the absolute local folder path for the workspace.
	Path string `json:"path"`
	// Name is the optional human-readable workspace name.
	Name string `json:"name"`
}

// sourceRequest is the decoded body for POST /workspace/source.
type sourceRequest struct {
	// WorkspaceID is a workspace path or ID.
	WorkspaceID string `json:"workspace_id"`
	// Connector is the connector name, e.g. github or jira.
	Connector string `json:"connector"`
	// SourceURI is the external source URI to save.
	SourceURI string `json:"source_uri"`
	// URI is accepted for frontend compatibility with existing source forms.
	URI string `json:"uri,omitempty"`
}

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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
