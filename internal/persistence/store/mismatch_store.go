package store

import (
	"context"
	"database/sql"
	"fmt"

	"context-os/domain/types"
)

// ─── Mismatches ───────────────────────────────────────────────────────────────

// MismatchStore is the PostgreSQL-backed MismatchRepository.
type MismatchStore struct {
	db *sql.DB
}

// NewMismatchStore returns a MismatchStore backed by the provided connection pool.
func NewMismatchStore(db *sql.DB) *MismatchStore {
	return &MismatchStore{db: db}
}

// UpsertMismatches persists mismatches, updating on conflict.
func (s *MismatchStore) UpsertMismatches(ctx context.Context, workspaceID string, mismatches []types.Mismatch, traceID string) error {
	for _, m := range mismatches {
		entityIDs := m.EntityIDs
		if entityIDs == nil {
			entityIDs = []string{}
		}
		evidence := m.Evidence
		if evidence == nil {
			evidence = []string{}
		}
		affectedRoles := m.AffectedRoles
		if affectedRoles == nil {
			affectedRoles = []string{}
		}
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO mismatches
				(id, workspace_id, mismatch_type, summary, entity_ids, severity,
				 confidence, impact, evidence, affected_roles, recommended, trace_id, detected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
			ON CONFLICT (id, workspace_id) DO UPDATE SET
				summary        = EXCLUDED.summary,
				confidence     = EXCLUDED.confidence,
				evidence       = EXCLUDED.evidence,
				recommended    = EXCLUDED.recommended,
				trace_id       = EXCLUDED.trace_id,
				detected_at    = NOW()
		`,
			m.ID, workspaceID, m.Type, m.Summary,
			aliasesToPGArray(entityIDs), m.Severity, m.Confidence,
			m.Impact, aliasesToPGArray(evidence),
			aliasesToPGArray(affectedRoles), m.Recommended, traceID,
		)
		if err != nil {
			return fmt.Errorf("store: upsert mismatch %s: %w", m.ID, err)
		}
	}
	return nil
}

// ListByWorkspace returns mismatches ordered by detected_at desc.
func (s *MismatchStore) ListByWorkspace(ctx context.Context, workspaceID, severityMin string, limit int) ([]types.Mismatch, error) {
	query := `SELECT id, mismatch_type, summary, entity_ids, severity,
	                 confidence, impact, evidence, affected_roles, recommended
	          FROM mismatches WHERE workspace_id = $1`
	args := []any{workspaceID}

	if severityMin != "" {
		args = append(args, severityMin)
		query += fmt.Sprintf(" AND severity = $%d", len(args))
	}
	query += " ORDER BY detected_at DESC"
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list mismatches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.Mismatch
	for rows.Next() {
		var m types.Mismatch
		var entityIDsRaw, evidenceRaw, affectedRolesRaw string
		if err := rows.Scan(
			&m.ID, &m.Type, &m.Summary, &entityIDsRaw, &m.Severity,
			&m.Confidence, &m.Impact, &evidenceRaw, &affectedRolesRaw, &m.Recommended,
		); err != nil {
			return nil, fmt.Errorf("store: scan mismatch: %w", err)
		}
		m.EntityIDs = parsePGArray(entityIDsRaw)
		m.Evidence = parsePGArray(evidenceRaw)
		m.AffectedRoles = parsePGArray(affectedRolesRaw)
		out = append(out, m)
	}
	return out, rows.Err()
}
