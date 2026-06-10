package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

// readBoundedBody reads and validates a JSON request body within a size limit.
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

// workspaceRefFromQuery extracts workspace scope from workspace_id or workspace_path query parameters.
func workspaceRefFromQuery(r *http.Request) string {
	if value := strings.TrimSpace(r.URL.Query().Get("workspace_id")); value != "" {
		return value
	}
	return strings.TrimSpace(r.URL.Query().Get("workspace_path"))
}

// resolveWorkspace finds a workspace by path, ID, or stored path fallback.
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

// writeResolveWorkspaceError maps workspace lookup errors to API responses.
func writeResolveWorkspaceError(w http.ResponseWriter, err error) {
	if os.IsNotExist(err) {
		response.WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
}

// firstNonEmpty returns the first non-blank string without altering its original spacing.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
