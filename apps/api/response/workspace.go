package response

import (
	"time"

	"context-os/domain/repository"
)

// Workspace is the JSON projection of a registered local workspace.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceList is the JSON payload returned by GET /workspace.
type WorkspaceList struct {
	Workspaces []Workspace `json:"workspaces"`
	Count      int         `json:"count"`
}

// WorkspaceSync is the JSON projection of connector sync state.
type WorkspaceSync struct {
	WorkspaceID  string     `json:"workspace_id"`
	Connector    string     `json:"connector"`
	SourceURI    string     `json:"source_uri"`
	Cursor       string     `json:"cursor,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	EventCount   int        `json:"event_count"`
	Status       string     `json:"status,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
}

// WorkspaceStatus is the JSON payload returned by GET /workspace/status.
type WorkspaceStatus struct {
	Workspace     Workspace       `json:"workspace"`
	EventCount    int             `json:"event_count"`
	EntityCount   int             `json:"entity_count"`
	MismatchCount int             `json:"mismatch_count"`
	Syncs         []WorkspaceSync `json:"syncs"`
}

// NewWorkspace maps a repository workspace into an API response.
func NewWorkspace(workspace repository.Workspace) Workspace {
	return Workspace{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Path:      workspace.Path,
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

// NewWorkspaceList maps repository workspaces into a list response.
func NewWorkspaceList(workspaces []repository.Workspace) WorkspaceList {
	items := make([]Workspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		items = append(items, NewWorkspace(workspace))
	}
	return WorkspaceList{Workspaces: items, Count: len(items)}
}

// NewWorkspaceStatus maps workspace status state into an API response.
func NewWorkspaceStatus(
	workspace repository.Workspace,
	eventCount int,
	entityCount int,
	mismatchCount int,
	syncs []repository.ConnectorSync,
) WorkspaceStatus {
	return WorkspaceStatus{
		Workspace:     NewWorkspace(workspace),
		EventCount:    eventCount,
		EntityCount:   entityCount,
		MismatchCount: mismatchCount,
		Syncs:         NewWorkspaceSyncs(syncs),
	}
}

// NewWorkspaceSyncs maps repository sync rows into API responses.
func NewWorkspaceSyncs(syncs []repository.ConnectorSync) []WorkspaceSync {
	items := make([]WorkspaceSync, 0, len(syncs))
	for _, sync := range syncs {
		items = append(items, WorkspaceSync{
			WorkspaceID:  sync.WorkspaceID,
			Connector:    sync.Connector,
			SourceURI:    sync.SourceURI,
			Cursor:       sync.Cursor,
			LastSyncedAt: sync.LastSyncedAt,
			EventCount:   sync.EventCount,
			Status:       sync.Status,
			LastError:    sync.LastError,
		})
	}
	return items
}