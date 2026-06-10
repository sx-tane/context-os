package store

// White-box tests inspect unexported db fields to verify constructor wiring.

import (
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewSyncStoreUsesProvidedDB verifies the sync store constructor keeps the provided database handle.
func TestNewSyncStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewSyncStore(db)

	if store.db != db {
		t.Fatalf("NewSyncStore().db = %p, want %p", store.db, db)
	}
}

// TestSyncStoreImplementsRepository verifies sync persistence supports the sync repository contract.
func TestSyncStoreImplementsRepository(t *testing.T) {
	t.Parallel()

	var _ repository.SyncRepository = (*SyncStore)(nil)
}
