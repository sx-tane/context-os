// Package artifacts provides HTTP handlers for querying local source artifacts.
package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

const artifactsRequestTimeout = 10 * time.Second

var (
	genericLiveEvidenceSourcePattern = regexp.MustCompile(`(?i)^(body|content|created_at|date|description|enum|field|fields|file|id|key|label|metadata|name|path|source|status|title|type|updated_at|uri|url|value)(/[a-z0-9_.-]+)?$`)
	jiraSourcePattern                = regexp.MustCompile(`^[a-z][a-z0-9]+-\d+$`)
	githubSourcePattern              = regexp.MustCompile(`^[a-z0-9_.-]+/[a-z0-9_.-]+$`)
)

// CleanupResult is the JSON payload returned by POST /artifacts/live-evidence/cleanup.
type CleanupResult struct {
	WorkspaceID               string   `json:"workspace_id"`
	WorkspacePath             string   `json:"workspace_path"`
	MatchedCount              int      `json:"matched_count"`
	DeletedCount              int      `json:"deleted_count"`
	DeletedIDs                []string `json:"deleted_ids"`
	DeletedGraphEntityCount   int      `json:"deleted_graph_entity_count,omitempty"`
	DeletedGraphRelationCount int      `json:"deleted_graph_relationship_count,omitempty"`
}

// DeleteRequest is the JSON body accepted by POST /artifacts/delete.
type DeleteRequest struct {
	WorkspaceID   string   `json:"workspace_id"`
	WorkspacePath string   `json:"workspace_path"`
	IDs           []string `json:"ids"`
}

// Handler holds repository dependencies for artifact HTTP handlers.
type Handler struct {
	workspaces repository.WorkspaceRepository
	events     repository.EventRepository
	graph      repository.GraphEvidenceDeleter
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(workspaces repository.WorkspaceRepository, events repository.EventRepository) *Handler {
	return &Handler{workspaces: workspaces, events: events}
}

// WithGraphEvidenceDeleter enables graph pruning for explicitly deleted artifact evidence.
func (h *Handler) WithGraphEvidenceDeleter(graph repository.GraphEvidenceDeleter) *Handler {
	h.graph = graph
	return h
}

// Query handles GET /artifacts.
//
// @Summary      Query local artifacts
// @Description  Returns workspace-scoped source artifacts filtered by connector, source, time range, and text.
// @Tags         artifacts
// @Produce      json
// @Param        workspace_id  query     string  true   "Workspace path or ID"
// @Param        connector     query     string  false  "Connector name"
// @Param        source_uri    query     string  false  "Source URI, channel, repository, or folder"
// @Param        q             query     string  false  "Text search"
// @Param        since         query     string  false  "RFC3339 inclusive lower bound"
// @Param        until         query     string  false  "RFC3339 exclusive upper bound"
// @Param        limit         query     int     false  "Maximum artifacts"
// @Success      200           {object}  response.ArtifactList
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /artifacts [get]
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	if h.workspaces == nil || h.events == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "artifact store is unavailable")
		return
	}

	workspaceRef := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if workspaceRef == "" {
		workspaceRef = strings.TrimSpace(r.URL.Query().Get("workspace_path"))
	}
	if workspaceRef == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), artifactsRequestTimeout)
	defer cancel()

	workspace, err := h.resolveWorkspace(ctx, workspaceRef)
	if err != nil {
		if err == sql.ErrNoRows {
			response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	query, err := buildEventQuery(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	events, err := h.events.Query(ctx, workspace.ID, query)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, response.ArtifactList{
		WorkspaceID:   workspace.ID,
		WorkspacePath: workspace.Path,
		Connector:     query.Connector,
		SourceURI:     query.SourceURI,
		Query:         query.Text,
		Count:         len(events),
		Artifacts:     response.NewArtifacts(events),
	})
}

// CleanupLiveEvidence handles POST /artifacts/live-evidence/cleanup.
//
// @Summary      Clean noisy live evidence
// @Description  Removes old noisy live_chat_answer artifacts for a workspace. This endpoint is explicit and never runs automatically.
// @Tags         artifacts
// @Produce      json
// @Param        workspace_id  query     string  true  "Workspace path or ID"
// @Success      200           {object}  CleanupResult
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /artifacts/live-evidence/cleanup [post]
func (h *Handler) CleanupLiveEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	if h.workspaces == nil || h.events == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "artifact store is unavailable")
		return
	}
	deleter, ok := h.events.(repository.EventDeleter)
	if !ok {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "artifact cleanup is unavailable")
		return
	}

	workspaceRef := workspaceRefFromRequest(r)
	if workspaceRef == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), artifactsRequestTimeout)
	defer cancel()

	workspace, err := h.resolveWorkspace(ctx, workspaceRef)
	if err != nil {
		if err == sql.ErrNoRows {
			response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	events, err := h.events.Query(ctx, workspace.ID, repository.EventQuery{Limit: 1000})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	ids := noisyLiveEvidenceIDs(events)
	deleted, err := deleter.DeleteByIDs(ctx, workspace.ID, ids)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	graphResult, err := h.deleteGraphEvidence(ctx, workspace.ID, ids)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, CleanupResult{
		WorkspaceID:               workspace.ID,
		WorkspacePath:             workspace.Path,
		MatchedCount:              len(ids),
		DeletedCount:              deleted,
		DeletedIDs:                ids,
		DeletedGraphEntityCount:   graphResult.DeletedEntityCount,
		DeletedGraphRelationCount: graphResult.DeletedRelationshipCount,
	})
}

// Delete handles POST /artifacts/delete.
//
// @Summary      Delete selected artifacts
// @Description  Explicitly removes user-selected workspace-scoped Activity artifacts by ID.
// @Tags         artifacts
// @Accept       json
// @Produce      json
// @Param        request  body      DeleteRequest  true  "Artifact IDs to delete"
// @Success      200      {object}  CleanupResult
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      405      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /artifacts/delete [post]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	if h.workspaces == nil || h.events == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "artifact store is unavailable")
		return
	}
	deleter, ok := h.events.(repository.EventDeleter)
	if !ok {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "artifact delete is unavailable")
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	workspaceRef := firstNonEmpty(strings.TrimSpace(req.WorkspaceID), strings.TrimSpace(req.WorkspacePath))
	if workspaceRef == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id is required")
		return
	}
	ids := cleanArtifactIDs(req.IDs)
	if len(ids) == 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "at least one artifact id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), artifactsRequestTimeout)
	defer cancel()

	workspace, err := h.resolveWorkspace(ctx, workspaceRef)
	if err != nil {
		if err == sql.ErrNoRows {
			response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	deleted, err := deleter.DeleteByIDs(ctx, workspace.ID, ids)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	graphResult, err := h.deleteGraphEvidence(ctx, workspace.ID, ids)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, CleanupResult{
		WorkspaceID:               workspace.ID,
		WorkspacePath:             workspace.Path,
		MatchedCount:              len(ids),
		DeletedCount:              deleted,
		DeletedIDs:                ids,
		DeletedGraphEntityCount:   graphResult.DeletedEntityCount,
		DeletedGraphRelationCount: graphResult.DeletedRelationshipCount,
	})
}

func (h *Handler) deleteGraphEvidence(ctx context.Context, workspaceID string, ids []string) (repository.GraphCleanupResult, error) {
	if h.graph == nil || len(ids) == 0 {
		return repository.GraphCleanupResult{}, nil
	}
	result, err := h.graph.DeleteGraphEvidenceByEventIDs(ctx, workspaceID, ids)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("artifacts: delete graph evidence: %w", err)
	}
	return result, nil
}

func (h *Handler) resolveWorkspace(ctx context.Context, ref string) (repository.Workspace, error) {
	workspace, err := h.workspaces.GetByPath(ctx, ref)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("artifacts: get workspace by path: %w", err)
	}
	if workspace != nil {
		return *workspace, nil
	}

	workspaces, err := h.workspaces.List(ctx)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("artifacts: list workspaces: %w", err)
	}
	for _, workspace := range workspaces {
		if workspace.ID == ref || workspace.Path == ref {
			return workspace, nil
		}
	}
	return repository.Workspace{}, sql.ErrNoRows
}

func buildEventQuery(r *http.Request) (repository.EventQuery, error) {
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		return repository.EventQuery{}, err
	}
	since, err := parseOptionalTime(r.URL.Query().Get("since"), r.URL.Query().Get("after"))
	if err != nil {
		return repository.EventQuery{}, err
	}
	until, err := parseOptionalTime(r.URL.Query().Get("until"), r.URL.Query().Get("before"))
	if err != nil {
		return repository.EventQuery{}, err
	}

	return repository.EventQuery{
		Connector: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("connector"))),
		SourceURI: strings.TrimSpace(r.URL.Query().Get("source_uri")),
		Text:      strings.TrimSpace(r.URL.Query().Get("q")),
		Since:     since,
		Until:     until,
		Limit:     limit,
	}, nil
}

func workspaceRefFromRequest(r *http.Request) string {
	workspaceRef := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if workspaceRef == "" {
		workspaceRef = strings.TrimSpace(r.URL.Query().Get("workspace_path"))
	}
	return workspaceRef
}

func cleanArtifactIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) > 500 {
		return out[:500]
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func noisyLiveEvidenceIDs(events []repository.IngestEvent) []string {
	ids := make([]string, 0)
	for _, event := range events {
		if isNoisyLiveEvidence(event) {
			ids = append(ids, event.ID)
		}
	}
	return ids
}

func isNoisyLiveEvidence(event repository.IngestEvent) bool {
	metadata := event.Metadata
	if metadata == nil || metadata["evidence_kind"] != "live_chat_answer" {
		return false
	}
	if strings.TrimSpace(metadata["source_label"]) != "" {
		return false
	}
	sourceURI := strings.ToLower(strings.TrimSpace(event.SourceURI))
	if sourceURI == "" || genericLiveEvidenceSourcePattern.MatchString(sourceURI) {
		return true
	}
	if strings.Count(metadata["related_sources"], ",") > 0 && !strings.Contains(event.Body, "Source:") {
		return true
	}
	if strings.Contains(sourceURI, "://") || strings.HasPrefix(sourceURI, "#") {
		return false
	}
	if event.Connector == "jira" && jiraSourcePattern.MatchString(sourceURI) {
		return false
	}
	if event.Connector == "github" && githubSourcePattern.MatchString(sourceURI) {
		return false
	}
	if strings.Contains(sourceURI, ".com/") || strings.Contains(sourceURI, ".net/") || strings.Contains(sourceURI, ".org/") {
		return true
	}
	if strings.Count(sourceURI, "/") > 0 && event.Connector != "github" {
		return true
	}
	return false
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if limit < 1 {
		return 0, fmt.Errorf("limit must be greater than zero")
	}
	if limit > 100 {
		return 100, nil
	}
	return limit, nil
}

func parseOptionalTime(values ...string) (*time.Time, error) {
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("time values must use RFC3339")
		}
		parsed = parsed.UTC()
		return &parsed, nil
	}
	return nil, nil
}
