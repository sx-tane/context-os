package store

import (
	"context"
	"database/sql"
	"fmt"

	"context-os/domain/repository"
)

// ─── Workspace UI State ─────────────────────────────────────────────────────

// WorkspaceUIStateStore is the PostgreSQL-backed WorkspaceUIStateRepository.
type WorkspaceUIStateStore struct {
	db *sql.DB
}

// NewWorkspaceUIStateStore returns a WorkspaceUIStateStore backed by the provided connection pool.
func NewWorkspaceUIStateStore(db *sql.DB) *WorkspaceUIStateStore {
	return &WorkspaceUIStateStore{db: db}
}

// Get returns a workspace UI state document by key.
func (s *WorkspaceUIStateStore) Get(ctx context.Context, workspaceID, stateKey string) (*repository.WorkspaceUIState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, state_key, payload_json, updated_at
		FROM workspace_ui_state
		WHERE workspace_id = $1 AND state_key = $2
	`, workspaceID, stateKey)
	var state repository.WorkspaceUIState
	if err := row.Scan(&state.WorkspaceID, &state.StateKey, &state.PayloadJSON, &state.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get workspace UI state: %w", err)
	}
	return &state, nil
}

// Put creates or replaces a workspace UI state document by key.
func (s *WorkspaceUIStateStore) Put(ctx context.Context, state repository.WorkspaceUIState) error {
	if len(state.PayloadJSON) == 0 {
		state.PayloadJSON = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_ui_state (workspace_id, state_key, payload_json, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (workspace_id, state_key) DO UPDATE SET
			payload_json = EXCLUDED.payload_json,
			updated_at   = EXCLUDED.updated_at
	`, state.WorkspaceID, state.StateKey, state.PayloadJSON)
	if err != nil {
		return fmt.Errorf("store: put workspace UI state: %w", err)
	}
	return nil
}
