// Package repository defines the domain repository interfaces for workspace-scoped persistence.
// Implementations live in internal/persistence/store.
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
	ID string
	// Name is the human-readable workspace name.
	Name string
	// Path is the absolute local folder path used as the project key.
	Path string
	// CreatedAt is the UTC timestamp when the workspace was first registered.
	CreatedAt time.Time
	// UpdatedAt is the UTC timestamp of the last write to the workspace row.
	UpdatedAt time.Time
}

// IngestEvent is the stored record for one raw source event captured by ingestion.
type IngestEvent struct {
	// ID is the replay-stable event identifier from the source connector.
	ID string
	// WorkspaceID links the event to its workspace.
	WorkspaceID string
	// Connector is the source connector name, e.g. "github" or "slack".
	Connector string
	// SourceURI is the URI of the source resource that produced the event.
	SourceURI string
	// EventType is the domain event type, e.g. "document.ingested".
	EventType string
	// Title is the trimmed subject line of the source artifact.
	Title string
	// Body is the trimmed content body of the source artifact.
	Body string
	// ContentHash is the SHA-256 of the normalised body used for deduplication.
	ContentHash string
	// Metadata holds key-value pairs carried from the source event.
	Metadata map[string]string
	// SchemaVersion is the event envelope schema version.
	SchemaVersion string
	// IngestedAt is the UTC time this event was persisted.
	IngestedAt time.Time
}

// EventQuery describes workspace-scoped filters for ingested source artifacts.
type EventQuery struct {
	// Connector filters artifacts to one connector when non-empty.
	Connector string
	// SourceURI filters artifacts to one channel, repository, folder, or source URI when non-empty.
	SourceURI string
	// Text filters artifacts whose title, body, or source URI contains the value when non-empty.
	Text string
	// Since filters artifacts ingested at or after this timestamp when set.
	Since *time.Time
	// Until filters artifacts ingested before this timestamp when set.
	Until *time.Time
	// Limit caps result count when greater than zero.
	Limit int
}

// ConnectorSync is the stored cursor and sync state for one connector in a workspace.
type ConnectorSync struct {
	// WorkspaceID links the record to its workspace.
	WorkspaceID string
	// Connector is the source connector name.
	Connector string
	// SourceURI is the URI this sync record covers.
	SourceURI string
	// Cursor is the replay checkpoint used for incremental sync.
	Cursor string
	// LastSyncedAt is when the last successful sync completed, nil if never.
	LastSyncedAt *time.Time
	// EventCount is the number of events ingested in the last sync run.
	EventCount int
	// Status is the current sync state: connected | idle | syncing | error.
	Status string
	// LastError is the last error message, empty if no error.
	LastError string
}

// WorkspaceUIState stores durable frontend workflow state for a workspace.
type WorkspaceUIState struct {
	// WorkspaceID links the state to its workspace.
	WorkspaceID string
	// StateKey identifies the UI state document, e.g. "analysis_basket".
	StateKey string
	// PayloadJSON is the raw JSON payload owned by the typed API handler.
	PayloadJSON []byte
	// UpdatedAt is the UTC timestamp of the last write.
	UpdatedAt time.Time
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

// WorkspaceResetter clears a workspace and all workspace-scoped memory rows.
type WorkspaceResetter interface {
	// DeleteByPath deletes a workspace by path and removes linked memory rows.
	// It is a no-op when the workspace does not exist.
	DeleteByPath(ctx context.Context, path string) error
}

// WorkspaceUIStateRepository manages durable workspace-scoped UI state.
type WorkspaceUIStateRepository interface {
	// Get returns the stored state for a workspace/key pair, or nil when absent.
	Get(ctx context.Context, workspaceID, stateKey string) (*WorkspaceUIState, error)
	// Put creates or replaces the stored state for a workspace/key pair.
	Put(ctx context.Context, state WorkspaceUIState) error
}

// EventRepository manages ingested source events.
type EventRepository interface {
	// UpsertBatch writes events, updating duplicates with the same (id, workspace_id).
	// Returns the number of rows inserted or updated.
	UpsertBatch(ctx context.Context, workspaceID string, events []IngestEvent) (int, error)
	// ListByWorkspace returns events for a workspace ordered by ingested_at desc.
	// If connector is non-empty, results are filtered by connector.
	ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]IngestEvent, error)
	// Query returns events for a workspace ordered by ingested_at desc using optional artifact filters.
	Query(ctx context.Context, workspaceID string, query EventQuery) ([]IngestEvent, error)
	// Count returns the total number of events for a workspace and optional connector.
	Count(ctx context.Context, workspaceID, connector string) (int, error)
}

// EventDeleter removes selected workspace-scoped source events.
type EventDeleter interface {
	// DeleteByIDs removes events by ID for one workspace and returns the deleted row count.
	DeleteByIDs(ctx context.Context, workspaceID string, ids []string) (int, error)
}

// GraphEvidenceDeleter removes graph rows tied to selected source event evidence.
type GraphEvidenceDeleter interface {
	// DeleteGraphEvidenceByEventIDs removes graph rows supported by the provided workspace event IDs.
	DeleteGraphEvidenceByEventIDs(ctx context.Context, workspaceID string, eventIDs []string) (GraphCleanupResult, error)
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
	// ListRelationships returns all relationships for a workspace, optionally scoped
	// to relationships touching one of the provided entity IDs.
	ListRelationships(ctx context.Context, workspaceID string, entityIDs []string) ([]types.Relationship, error)
}

// GraphCleanupResult reports explicit workspace-scoped graph cleanup counts.
type GraphCleanupResult struct {
	// MatchedEntityCount is the number of backend-classified noisy entity rows found.
	MatchedEntityCount int
	// DeletedEntityCount is the number of backend-classified noisy entity rows removed.
	DeletedEntityCount int
	// MatchedRelationshipCount is the number of low-signal or dangling relationship rows found.
	MatchedRelationshipCount int
	// DeletedRelationshipCount is the number of low-signal or dangling relationship rows removed.
	DeletedRelationshipCount int
}

// GraphNoiseCleaner permanently removes backend-classified low-signal graph rows.
type GraphNoiseCleaner interface {
	// CleanupGraphNoise removes only workspace-scoped rows classified as graph noise.
	CleanupGraphNoise(ctx context.Context, workspaceID string) (GraphCleanupResult, error)
}

// GraphEntityDeleter permanently removes one graph entity and its relationships.
type GraphEntityDeleter interface {
	// DeleteGraphEntity removes one workspace-scoped entity and relationships touching it.
	DeleteGraphEntity(ctx context.Context, workspaceID, entityID string) (GraphCleanupResult, error)
}

// RelationshipCounter reports relationship density for a workspace.
type RelationshipCounter interface {
	// CountRelationships returns the total relationship count for a workspace.
	CountRelationships(ctx context.Context, workspaceID string) (int, error)
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

// AuditEvent is a record of a significant action performed against a workspace.
type AuditEvent struct {
	// WorkspaceID is the workspace this audit event belongs to.
	WorkspaceID string
	// EventType identifies the action, e.g. "workspace.registered" or "ingest.completed".
	EventType string
	// Actor is the originator of the action, e.g. a user ID or service name.
	Actor string
	// Connector is the involved connector name, empty when not applicable.
	Connector string
	// SourceURI is the ingested URI, empty when not applicable.
	SourceURI string
	// EntityID is the affected entity, empty when not applicable.
	EntityID string
	// TraceID links the audit record to a pipeline run trace.
	TraceID string
	// Payload carries additional key-value context about the action.
	Payload map[string]string
}

// AuditRepository appends immutable audit log entries.
type AuditRepository interface {
	// Log appends an audit event to the audit log table.
	Log(ctx context.Context, e AuditEvent) error
}

// AuditCounter reports audit row counts for a workspace.
type AuditCounter interface {
	// CountByWorkspace returns the total audit rows for a workspace.
	CountByWorkspace(ctx context.Context, workspaceID string) (int, error)
}
