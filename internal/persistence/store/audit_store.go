package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"context-os/domain/repository"
)

// ─── Audit ────────────────────────────────────────────────────────────────────

// AuditStore is the PostgreSQL-backed AuditRepository.
type AuditStore struct {
	db *sql.DB
}

// NewAuditStore returns an AuditStore backed by the provided connection pool.
func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Log appends an audit event to the audit_log table.
func (s *AuditStore) Log(ctx context.Context, e repository.AuditEvent) error {
	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("store: marshal audit payload: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_log
			(workspace_id, event_type, actor, connector, source_uri, entity_id, trace_id, payload, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		e.WorkspaceID, e.EventType, e.Actor, e.Connector,
		e.SourceURI, e.EntityID, e.TraceID, payloadJSON,
	)
	if err != nil {
		return fmt.Errorf("store: log audit event: %w", err)
	}
	return nil
}

// CountByWorkspace returns the total audit rows for a workspace.
func (s *AuditStore) CountByWorkspace(ctx context.Context, workspaceID string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_log WHERE workspace_id = $1
	`, workspaceID).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count audit rows: %w", err)
	}
	return n, nil
}
