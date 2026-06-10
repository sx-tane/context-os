package store

// White-box tests inspect unexported db fields and zero-input guards without opening a database.

import (
	"context"
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewMismatchStoreUsesProvidedDB verifies the mismatch store constructor keeps the provided database handle.
func TestNewMismatchStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewMismatchStore(db)

	if store.db != db {
		t.Fatalf("NewMismatchStore().db = %p, want %p", store.db, db)
	}
}

// TestMismatchStoreImplementsRepository verifies mismatch persistence supports the mismatch repository contract.
func TestMismatchStoreImplementsRepository(t *testing.T) {
	t.Parallel()

	var _ repository.MismatchRepository = (*MismatchStore)(nil)
}

// TestMismatchStoreZeroInputUpsertAvoidsDatabase verifies empty mismatch writes return without touching the database.
func TestMismatchStoreZeroInputUpsertAvoidsDatabase(t *testing.T) {
	t.Parallel()

	store := NewMismatchStore(nil)
	if err := store.UpsertMismatches(context.Background(), "workspace", nil, "trace"); err != nil {
		t.Fatalf("UpsertMismatches() error = %v", err)
	}
}
