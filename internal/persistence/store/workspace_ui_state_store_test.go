package store

// White-box tests inspect unexported db fields to verify constructor wiring.

import (
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewWorkspaceUIStateStoreUsesProvidedDB verifies the workspace UI state store constructor keeps the provided database handle.
func TestNewWorkspaceUIStateStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewWorkspaceUIStateStore(db)

	if store.db != db {
		t.Fatalf("NewWorkspaceUIStateStore().db = %p, want %p", store.db, db)
	}
}

// TestWorkspaceUIStateStoreImplementsRepository verifies UI state persistence supports the workspace UI state repository contract.
func TestWorkspaceUIStateStoreImplementsRepository(t *testing.T) {
	t.Parallel()

	var _ repository.WorkspaceUIStateRepository = (*WorkspaceUIStateStore)(nil)
}
