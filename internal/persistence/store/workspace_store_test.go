package store

// White-box tests inspect unexported db fields and package-local cleanup table configuration.

import (
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewWorkspaceStoreUsesProvidedDB verifies the workspace store constructor keeps the provided database handle.
func TestNewWorkspaceStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewWorkspaceStore(db)

	if store.db != db {
		t.Fatalf("NewWorkspaceStore().db = %p, want %p", store.db, db)
	}
}

// TestWorkspaceStoreImplementsRepositories verifies workspace persistence supports the workspace repository contracts.
func TestWorkspaceStoreImplementsRepositories(t *testing.T) {
	t.Parallel()

	var _ repository.WorkspaceRepository = (*WorkspaceStore)(nil)
	var _ repository.WorkspaceResetter = (*WorkspaceStore)(nil)
}
