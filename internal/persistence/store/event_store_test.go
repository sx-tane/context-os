package store

// White-box tests inspect unexported db fields and zero-input guards without opening a database.

import (
	"context"
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewEventStoreUsesProvidedDB verifies the event store constructor keeps the provided database handle.
func TestNewEventStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewEventStore(db)

	if store.db != db {
		t.Fatalf("NewEventStore().db = %p, want %p", store.db, db)
	}
}

// TestEventStoreImplementsRepositories verifies event persistence supports query and delete repository contracts.
func TestEventStoreImplementsRepositories(t *testing.T) {
	t.Parallel()

	var _ repository.EventRepository = (*EventStore)(nil)
	var _ repository.EventDeleter = (*EventStore)(nil)
}

// TestEventStoreZeroInputGuardsAvoidDatabase verifies empty write and delete calls return without touching the database.
func TestEventStoreZeroInputGuardsAvoidDatabase(t *testing.T) {
	t.Parallel()

	store := NewEventStore(nil)
	written, err := store.UpsertBatch(context.Background(), "workspace", nil)
	if err != nil {
		t.Fatalf("UpsertBatch() error = %v", err)
	}
	if written != 0 {
		t.Fatalf("UpsertBatch() written = %d, want 0", written)
	}

	deleted, err := store.DeleteByIDs(context.Background(), "workspace", nil)
	if err != nil {
		t.Fatalf("DeleteByIDs() error = %v", err)
	}
	if deleted != 0 {
		t.Fatalf("DeleteByIDs() deleted = %d, want 0", deleted)
	}
}
