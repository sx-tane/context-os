// Package workspace provides HTTP handlers for ContextOS workspace management.
package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/repository"
	internalchat "context-os/internal/runtime/chat"
)

const workspaceRequestTimeout = 10 * time.Second

// Handler holds the repository dependencies for workspace HTTP handlers.
type Handler struct {
	workspaces  repository.WorkspaceRepository
	events      repository.EventRepository
	entities    repository.EntityRepository
	mismatches  repository.MismatchRepository
	connSync    repository.SyncRepository
	uiState     repository.WorkspaceUIStateRepository
	audit       repository.AuditRepository
	parsedDir   string
	snapshotDir string
	sessionDir  string
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

// WithUIStateRepository attaches durable frontend workflow state handlers.
func (h *Handler) WithUIStateRepository(uiState repository.WorkspaceUIStateRepository) *Handler {
	h.uiState = uiState
	return h
}

// WithLocalArtifactDirs configures workspace-scoped local JSON cleanup.
func (h *Handler) WithLocalArtifactDirs(parsedDir, snapshotDir string) *Handler {
	h.parsedDir = parsedDir
	h.snapshotDir = snapshotDir
	return h
}

// WithCodexChatSessionDir configures workspace-scoped Codex chat session metadata cleanup.
func (h *Handler) WithCodexChatSessionDir(sessionDir string) *Handler {
	h.sessionDir = sessionDir
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

func (h *Handler) deleteLocalWorkspaceArtifacts(workspaceID string) error {
	if workspaceID == "" {
		return nil
	}
	if h.parsedDir != "" {
		if err := os.RemoveAll(filepath.Join(h.parsedDir, workspaceID)); err != nil {
			return fmt.Errorf("delete parsed workspace artifacts: %w", err)
		}
	}
	if h.snapshotDir == "" {
		return h.deleteCodexChatSession(workspaceID)
	}
	if err := removeSnapshotFiles(h.snapshotDir, workspaceID+".json"); err != nil {
		return err
	}
	if err := removeSnapshotFiles(h.snapshotDir, workspaceID+"_*.json"); err != nil {
		return err
	}
	return h.deleteCodexChatSession(workspaceID)
}

func (h *Handler) deleteCodexChatSession(workspaceID string) error {
	if h.sessionDir == "" {
		return nil
	}
	if err := internalchat.NewCodexSessionStore(h.sessionDir).Delete(workspaceID); err != nil {
		return fmt.Errorf("delete codex chat session metadata: %w", err)
	}
	return nil
}

func removeSnapshotFiles(dir, pattern string) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("find snapshot artifacts: %w", err)
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete snapshot artifact %q: %w", match, err)
		}
	}
	return nil
}

func workspaceIDForCleanup(path string, workspace *repository.Workspace) string {
	if workspace != nil && strings.TrimSpace(workspace.ID) != "" {
		return workspace.ID
	}
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.TrimPrefix(id, "_")
	if id == "" {
		return "workspace"
	}
	return id
}

func readBoundedBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid JSON")
	}
	return body, nil
}

func workspaceRefFromQuery(r *http.Request) string {
	if value := strings.TrimSpace(r.URL.Query().Get("workspace_id")); value != "" {
		return value
	}
	return strings.TrimSpace(r.URL.Query().Get("workspace_path"))
}

func (h *Handler) resolveWorkspace(ctx context.Context, ref string) (*repository.Workspace, error) {
	workspace, err := h.workspaces.GetByPath(ctx, ref)
	if err != nil {
		return nil, err
	}
	if workspace != nil {
		return workspace, nil
	}
	workspaces, err := h.workspaces.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, workspace := range workspaces {
		if workspace.ID == ref || workspace.Path == ref {
			found := workspace
			return &found, nil
		}
	}
	return nil, os.ErrNotExist
}

func writeResolveWorkspaceError(w http.ResponseWriter, err error) {
	if os.IsNotExist(err) {
		response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
}

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
		case "open", "checking", "done":
		default:
			return "", nil, fmt.Errorf("actions[%d].status must be open, checking, or done", index)
		}
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	return workspaceID, normalized, nil
}

func emptyAnalysisBasket(workspaceID string) any {
	return request.AnalysisBasket{WorkspaceID: workspaceID, Items: []request.AnalysisBasketItem{}}
}

func emptyFindingActions(workspaceID string) any {
	return request.FindingActions{WorkspaceID: workspaceID, Actions: []request.FindingActionItem{}}
}

// AnalysisBasket handles GET/PUT /workspace/analysis-basket.
//
// @Summary      Read or save the workspace analysis basket
// @Description  Persists selected analysis evidence for one workspace.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        workspace_id  query     string  false  "Workspace path or ID"
// @Param        body          body      request.AnalysisBasket  false  "Analysis basket payload"
// @Success      200           {object}  request.AnalysisBasket
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      503           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /workspace/analysis-basket [get]
// @Router       /workspace/analysis-basket [put]
func (h *Handler) AnalysisBasket(w http.ResponseWriter, r *http.Request) {
	h.workspaceUIState(w, r, "analysis_basket", emptyAnalysisBasket, validateAnalysisBasketPayload)
}

// FindingActions handles GET/PUT /workspace/finding-actions.
//
// @Summary      Read or save the workspace finding action checklist
// @Description  Persists finding action statuses for one workspace.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        workspace_id  query     string  false  "Workspace path or ID"
// @Param        body          body      request.FindingActions  false  "Finding actions payload"
// @Success      200           {object}  request.FindingActions
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      503           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /workspace/finding-actions [get]
// @Router       /workspace/finding-actions [put]
func (h *Handler) FindingActions(w http.ResponseWriter, r *http.Request) {
	h.workspaceUIState(w, r, "finding_actions", emptyFindingActions, validateFindingActionsPayload)
}

func (h *Handler) workspaceUIState(
	w http.ResponseWriter,
	r *http.Request,
	stateKey string,
	empty func(string) any,
	validate func([]byte) (string, []byte, error),
) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
		return
	}
	if h.uiState == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "state_unavailable", "workspace UI state is unavailable for this store")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	if r.Method == http.MethodGet {
		workspaceRef := workspaceRefFromQuery(r)
		if workspaceRef == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id query parameter is required")
			return
		}
		workspace, err := h.resolveWorkspace(ctx, workspaceRef)
		if err != nil {
			writeResolveWorkspaceError(w, err)
			return
		}
		state, err := h.uiState.Get(ctx, workspace.ID, stateKey)
		if err != nil {
			response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		if state == nil {
			response.WriteJSON(w, http.StatusOK, empty(workspace.Path))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(state.PayloadJSON)
		return
	}

	body, err := readBoundedBody(w, r, 1<<20)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	workspaceRef, payloadJSON, err := validate(body)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	workspace, err := h.resolveWorkspace(ctx, workspaceRef)
	if err != nil {
		writeResolveWorkspaceError(w, err)
		return
	}
	if err := h.uiState.Put(ctx, repository.WorkspaceUIState{
		WorkspaceID: workspace.ID,
		StateKey:    stateKey,
		PayloadJSON: payloadJSON,
	}); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payloadJSON)
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
