// Package repository defines the domain repository interfaces for workspace-scoped persistence.
// Implementations live in internal/store.
package repository

import (
	"context"
	"time"

	"context-os/domain/entities"
	"context-os/domain/types"
)

// Workspace is the stored record for a ContextOS workspace.
type Workspace struct {
	// ID is the primary key derived from the workspace path.
	ID        string
	// Name is the human-readable workspace name.
	Name      string
	// Path is the absolute local folder path used as the project key.
	Path      string
	// CreatedAt is the UTC timestamp when the workspace was first registered.
	CreatedAt time.Time
	// UpdatedAt is the UTC timestamp of the last write to the workspace row.
	UpdatedAt time.Time
}

// IngestEvent is the stored record for one raw source event captured by ingestion.
type IngestEvent struct {
	// ID is the replay-stable event identifier from the source connector.
	ID            string
	// WorkspaceID links the event to its workspace.
	WorkspaceID   string
	// Connector is the source connector name, e.g. "github" or "slack".
	Connector     string
	// SourceURI is the URI of the source resource that produced the event.
	SourceURI     string
	// EventType is the domain event type, e.g. "document.ingested".
	EventType     string
	// Title is the trimmed subject line of the source artifact.
	Title         string
	// Body is the trimmed content body of the source artifact.
	Body          string
	// ContentHash is the SHA-256 of the normalised body used for deduplication.
	ContentHash   string
	// Metadata holds key-value pairs carried from the source event.
	Metadata      map[string]string
	// SchemaVersion is the event envelope schema version.
	SchemaVersion string
	// IngestedAt is the UTC time this event was persisted.
	IngestedAt    time.Time
}

// ConnectorSync is the stored cursor and sync state for one connector in a workspace.
type ConnectorSync struct {
	// WorkspaceID links the record to its workspace.
	WorkspaceID   string
	// Connector is the source connector name.
	Connector     string
	// SourceURI is the URI this sync record covers.
	SourceURI     string
	// Cursor is the replay checkpoint used for incremental sync.
	Cursor        string
	// LastSyncedAt is when the last successful sync completed, nil if never.
	LastSyncedAt  *time.Time
	// EventCount is the number of events ingested in the last sync run.
	EventCount    int
	// Status is the current sync state: idle | syncing | error.
	Status        string
	// LastError is the last error message, empty if no error.
	LastError     string
}

// WorkspaceRepository manages workspace records.
type WorkspaceRepository interface {
	// Upsert creates or updates a workspace by its path. Returns the stored workspace.
	Upsert(ctx context.Context, w Workspace) (Workspace, error)
	// GetByPath retrieves a workspace by its absolute path. Returns nil, nil when not found.
	GetByPath(ctx context.Context, path string) (*Workspace, error)
	// List returns all registered workspaces ordered by created_at desc.
	List(ctx context.Context) ([]Workspace, error)
}

// EventRepository manages ingested source events.
type EventRepository interface {
	// UpsertBatch writes events, ignoring duplicates with the same (id, workspace_id).
	// Returns the number of new events written.
	UpsertBatch(ctx context.Context, workspaceID string, events []IngestEvent) (int, error)
	// ListByWorkspace returns events for a workspace ordered by ingested_at desc.
	// If connector is non-empty, results are filtered by connector.
	ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]IngestEvent, error)
	// Count returns the total number of events for a workspace and optional connector.
	Count(ctx context.Context, workspaceID, connector string) (int, error)
}

// EntityRepository manages canonical entities and their relationships.
type EntityRepository interface {
	// UpsertEntities persists canonical entities, updating confidence and aliases
	// when a record with the same (id, workspace_id) already exists.
	UpsertEntities(ctx context.Context, workspaceID string, canonical []entities.CanonicalEntity) error
	// UpsertRelationships persists relationships, updating confidence and evidence
	// when a record with the same (id, workspace_id) already exists.
	UpsertRelationships(ctx context.Context, workspaceID string, rels []types.Relationship) error
	// ListEntities returns all entities for a workspace, optionally filtered by entityType.
	ListEntities(ctx context.Context, workspaceID, entityType string) ([]entities.CanonicalEntity, error)
}

// MismatchRepository manages reasoning findings.
type MismatchRepository interface {
	// UpsertMismatches persists mismatches, updating when the same (id, workspace_id) already exists.
	UpsertMismatches(ctx context.Context, workspaceID string, mismatches []types.Mismatch, traceID string) error
	// ListByWorkspace returns mismatches ordered by detected_at desc.
	// If severityMin is non-empty ("low"|"medium"|"high"), results are filtered.
	ListByWorkspace(ctx context.Context, workspaceID, severityMin string, limit int) ([]types.Mismatch, error)
}

// SyncRepository manages connector sync cursors.
type SyncRepository interface {
	// Upsert writes or updates the sync state for one connector+URI in a workspace.
	Upsert(ctx context.Context, s ConnectorSync) error
	// Get returns the sync state for a connector+URI pair, or nil if not found.
	Get(ctx context.Context, workspaceID, connector, sourceURI string) (*ConnectorSync, error)
	// ListByWorkspace returns all connector syncs for a workspace.
	ListByWorkspace(ctx context.Context, workspaceID string) ([]ConnectorSync, error)
}
