package store

// White-box tests inspect unexported db fields and zero-input guards without opening a database.

import (
	"context"
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewEntityStoreUsesProvidedDB verifies the entity store constructor keeps the provided database handle.
func TestNewEntityStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewEntityStore(db)

	if store.db != db {
		t.Fatalf("NewEntityStore().db = %p, want %p", store.db, db)
	}
}

// TestEntityStoreImplementsRepositories verifies entity persistence supports graph repository contracts.
func TestEntityStoreImplementsRepositories(t *testing.T) {
	t.Parallel()

	var _ repository.EntityRepository = (*EntityStore)(nil)
	var _ repository.RelationshipCounter = (*EntityStore)(nil)
}

// TestEntityStoreZeroInputUpsertsAvoidDatabase verifies empty entity and relationship writes return without touching the database.
func TestEntityStoreZeroInputUpsertsAvoidDatabase(t *testing.T) {
	t.Parallel()

	store := NewEntityStore(nil)
	if err := store.UpsertEntities(context.Background(), "workspace", nil); err != nil {
		t.Fatalf("UpsertEntities() error = %v", err)
	}
	if err := store.UpsertRelationships(context.Background(), "workspace", nil); err != nil {
		t.Fatalf("UpsertRelationships() error = %v", err)
	}
}
