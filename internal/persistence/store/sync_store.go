package store

import (
	"context"
	"database/sql"
	"fmt"

	"context-os/domain/repository"
)

// ─── Sync ─────────────────────────────────────────────────────────────────────

// SyncStore is the PostgreSQL-backed SyncRepository.
type SyncStore struct {
	db *sql.DB
}

// NewSyncStore returns a SyncStore backed by the provided connection pool.
func NewSyncStore(db *sql.DB) *SyncStore {
	return &SyncStore{db: db}
}

// Upsert writes or updates the sync state for one connector+URI in a workspace.
func (s *SyncStore) Upsert(ctx context.Context, sync repository.ConnectorSync) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO connector_syncs
			(workspace_id, connector, source_uri, cursor, last_synced_at, event_count, status, last_error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (workspace_id, connector, source_uri) DO UPDATE SET
			cursor         = EXCLUDED.cursor,
			last_synced_at = EXCLUDED.last_synced_at,
			event_count    = EXCLUDED.event_count,
			status         = EXCLUDED.status,
			last_error     = EXCLUDED.last_error
	`,
		sync.WorkspaceID, sync.Connector, sync.SourceURI,
		sync.Cursor, sync.LastSyncedAt, sync.EventCount, sync.Status, sync.LastError,
	)
	if err != nil {
		return fmt.Errorf("store: upsert connector sync: %w", err)
	}
	return nil
}

// Get returns the sync state for a connector+URI pair, or nil if not found.
func (s *SyncStore) Get(ctx context.Context, workspaceID, connector, sourceURI string) (*repository.ConnectorSync, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, connector, source_uri, cursor, last_synced_at,
		       event_count, status, last_error
		FROM connector_syncs
		WHERE workspace_id = $1 AND connector = $2 AND source_uri = $3
	`, workspaceID, connector, sourceURI)

	var sync repository.ConnectorSync
	if err := row.Scan(
		&sync.WorkspaceID, &sync.Connector, &sync.SourceURI,
		&sync.Cursor, &sync.LastSyncedAt, &sync.EventCount, &sync.Status, &sync.LastError,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get connector sync: %w", err)
	}
	return &sync, nil
}

// ListByWorkspace returns all connector syncs for a workspace.
func (s *SyncStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT workspace_id, connector, source_uri, cursor, last_synced_at,
		       event_count, status, last_error
		FROM connector_syncs WHERE workspace_id = $1
		ORDER BY last_synced_at DESC NULLS LAST
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list connector syncs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.ConnectorSync
	for rows.Next() {
		var sync repository.ConnectorSync
		if err := rows.Scan(
			&sync.WorkspaceID, &sync.Connector, &sync.SourceURI,
			&sync.Cursor, &sync.LastSyncedAt, &sync.EventCount, &sync.Status, &sync.LastError,
		); err != nil {
			return nil, fmt.Errorf("store: scan connector sync: %w", err)
		}
		out = append(out, sync)
	}
	return out, rows.Err()
}
