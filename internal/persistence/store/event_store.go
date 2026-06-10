package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"context-os/domain/repository"

	"github.com/lib/pq"
)

// ─── Events ──────────────────────────────────────────────────────────────────

// EventStore is the PostgreSQL-backed EventRepository.
type EventStore struct {
	db *sql.DB
}

// NewEventStore returns an EventStore backed by the provided connection pool.
func NewEventStore(db *sql.DB) *EventStore {
	return &EventStore{db: db}
}

// UpsertBatch writes events, updating any with a duplicate (id, workspace_id).
// Returns the number of rows inserted or updated.
func (s *EventStore) UpsertBatch(ctx context.Context, workspaceID string, evts []repository.IngestEvent) (int, error) {
	if len(evts) == 0 {
		return 0, nil
	}
	var written int
	for _, e := range evts {
		if e.IngestedAt.IsZero() {
			e.IngestedAt = time.Now().UTC()
		}
		metaJSON, err := json.Marshal(e.Metadata)
		if err != nil {
			return written, fmt.Errorf("store: marshal metadata for event %s: %w", e.ID, err)
		}
		r, err := s.db.ExecContext(ctx, `
				INSERT INTO ingest_events
					(id, workspace_id, connector, source_uri, event_type, title, body,
					 content_hash, metadata, schema_version, ingested_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
				ON CONFLICT (id, workspace_id) DO UPDATE SET
					connector      = EXCLUDED.connector,
					source_uri     = EXCLUDED.source_uri,
					event_type     = EXCLUDED.event_type,
					title          = EXCLUDED.title,
					body           = EXCLUDED.body,
					content_hash   = EXCLUDED.content_hash,
					metadata       = EXCLUDED.metadata,
					schema_version = EXCLUDED.schema_version,
					ingested_at    = EXCLUDED.ingested_at
			`,
			e.ID, workspaceID, e.Connector, e.SourceURI, e.EventType,
			e.Title, e.Body, e.ContentHash, metaJSON, e.SchemaVersion, e.IngestedAt,
		)
		if err != nil {
			return written, fmt.Errorf("store: upsert event %s: %w", e.ID, err)
		}
		n, _ := r.RowsAffected()
		written += int(n)
	}
	return written, nil
}

// ListByWorkspace returns events for a workspace ordered by ingested_at desc.
func (s *EventStore) ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]repository.IngestEvent, error) {
	return s.Query(ctx, workspaceID, repository.EventQuery{
		Connector: connector,
		Limit:     limit,
	})
}

// Query returns events for a workspace ordered by ingested_at desc using optional artifact filters.
func (s *EventStore) Query(ctx context.Context, workspaceID string, eventQuery repository.EventQuery) ([]repository.IngestEvent, error) {
	query := `SELECT id, workspace_id, connector, source_uri, event_type, title, body,
	                 content_hash, metadata, schema_version, ingested_at
	          FROM ingest_events
	          WHERE workspace_id = $1`
	args := []any{workspaceID}

	if eventQuery.Connector != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(eventQuery.Connector)))
		query += fmt.Sprintf(" AND connector = $%d", len(args))
	}
	if eventQuery.SourceURI != "" {
		args = append(args, strings.TrimSpace(eventQuery.SourceURI))
		query += fmt.Sprintf(" AND source_uri = $%d", len(args))
	}
	if eventQuery.Since != nil {
		args = append(args, eventQuery.Since.UTC())
		query += fmt.Sprintf(" AND ingested_at >= $%d", len(args))
	}
	if eventQuery.Until != nil {
		args = append(args, eventQuery.Until.UTC())
		query += fmt.Sprintf(" AND ingested_at < $%d", len(args))
	}
	if eventQuery.Text != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(eventQuery.Text))+"%")
		query += fmt.Sprintf(" AND (LOWER(title) LIKE $%d OR LOWER(body) LIKE $%d OR LOWER(source_uri) LIKE $%d)", len(args), len(args), len(args))
	}
	query += " ORDER BY ingested_at DESC"
	if eventQuery.Limit > 0 {
		args = append(args, eventQuery.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.IngestEvent
	for rows.Next() {
		var e repository.IngestEvent
		var metaJSON []byte
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.Connector, &e.SourceURI,
			&e.EventType, &e.Title, &e.Body, &e.ContentHash,
			&metaJSON, &e.SchemaVersion, &e.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan event: %w", err)
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("store: unmarshal event metadata: %w", err)
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteByIDs removes workspace-scoped ingest events by ID and returns the deleted row count.
func (s *EventStore) DeleteByIDs(ctx context.Context, workspaceID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM ingest_events
		WHERE workspace_id = $1 AND id = ANY($2)
	`, workspaceID, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("store: delete events by id: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// Count returns the total number of events for a workspace and optional connector.
func (s *EventStore) Count(ctx context.Context, workspaceID, connector string) (int, error) {
	query := `SELECT COUNT(*) FROM ingest_events WHERE workspace_id = $1`
	args := []any{workspaceID}
	if connector != "" {
		args = append(args, connector)
		query += fmt.Sprintf(" AND connector = $%d", len(args))
	}
	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count events: %w", err)
	}
	return n, nil
}
