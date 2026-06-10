package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"context-os/domain/repository"
)

var workspaceScopedMemoryTables = []string{
	"audit_log",
	"workspace_ui_state",
	"connector_syncs",
	"mismatches",
	"relationships",
	"entities",
	"ingest_events",
}

// ─── Workspace ───────────────────────────────────────────────────────────────

// WorkspaceStore is the PostgreSQL-backed WorkspaceRepository.
type WorkspaceStore struct {
	db *sql.DB
}

// NewWorkspaceStore returns a WorkspaceStore backed by the provided connection pool.
func NewWorkspaceStore(db *sql.DB) *WorkspaceStore {
	return &WorkspaceStore{db: db}
}

// Upsert creates or updates a workspace record identified by its path.
func (s *WorkspaceStore) Upsert(ctx context.Context, w repository.Workspace) (repository.Workspace, error) {
	now := time.Now().UTC()
	if w.ID == "" {
		w.ID = workspaceIDFromPath(w.Path)
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	w.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (path) DO UPDATE SET
			name       = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`, w.ID, w.Name, w.Path, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("store: upsert workspace: %w", err)
	}
	return w, nil
}

// GetByPath retrieves a workspace by its absolute path. Returns nil, nil when not found.
func (s *WorkspaceStore) GetByPath(ctx context.Context, path string) (*repository.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, path, created_at, updated_at
		FROM workspaces WHERE path = $1
	`, path)
	var w repository.Workspace
	if err := row.Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt, &w.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get workspace by path: %w", err)
	}
	return &w, nil
}

// List returns all registered workspaces ordered by created_at desc.
func (s *WorkspaceStore) List(ctx context.Context) ([]repository.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, path, created_at, updated_at
		FROM workspaces ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list workspaces: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.Workspace
	for rows.Next() {
		var w repository.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan workspace: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// DeleteByPath deletes a workspace by path and removes all workspace-scoped memory rows.
func (s *WorkspaceStore) DeleteByPath(ctx context.Context, path string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin delete workspace by path: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var workspaceID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM workspaces WHERE path = $1
	`, path).Scan(&workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("store: find workspace by path for delete: %w", err)
	}

	for _, table := range workspaceScopedMemoryTables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE workspace_id = $1", table), workspaceID); err != nil {
			return fmt.Errorf("store: delete %s for workspace: %w", table, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM workspaces WHERE id = $1
	`, workspaceID); err != nil {
		return fmt.Errorf("store: delete workspace row by path: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit delete workspace by path: %w", err)
	}
	return nil
}
