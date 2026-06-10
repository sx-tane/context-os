package store

// White-box tests inspect unexported db fields to verify constructor wiring.

import (
	"database/sql"
	"testing"

	"context-os/domain/repository"
)

// TestNewAuditStoreUsesProvidedDB verifies the audit store constructor keeps the provided database handle.
func TestNewAuditStoreUsesProvidedDB(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	store := NewAuditStore(db)

	if store.db != db {
		t.Fatalf("NewAuditStore().db = %p, want %p", store.db, db)
	}
}

// TestAuditStoreImplementsRepositories verifies audit persistence supports audit logging and counting contracts.
func TestAuditStoreImplementsRepositories(t *testing.T) {
	t.Parallel()

	var _ repository.AuditRepository = (*AuditStore)(nil)
	var _ repository.AuditCounter = (*AuditStore)(nil)
}
