// Package artifacts provides HTTP handlers for querying local source artifacts.
package artifacts

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

const artifactsRequestTimeout = 10 * time.Second

// Handler holds repository dependencies for artifact HTTP handlers.
type Handler struct {
	workspaces repository.WorkspaceRepository
	events     repository.EventRepository
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(workspaces repository.WorkspaceRepository, events repository.EventRepository) *Handler {
	return &Handler{workspaces: workspaces, events: events}
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
