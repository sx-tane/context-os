package store

// White-box tests exercise graph cleanup zero-input guards without opening a database.

import (
	"context"
	"reflect"
	"testing"

	"context-os/domain/repository"
)

// TestGraphCleanupInterfaces verifies entity store exposes the graph cleanup repository contracts.
func TestGraphCleanupInterfaces(t *testing.T) {
	t.Parallel()

	var _ repository.GraphEvidenceDeleter = (*EntityStore)(nil)
	var _ repository.GraphNoiseCleaner = (*EntityStore)(nil)
	var _ repository.GraphEntityDeleter = (*EntityStore)(nil)
}

// TestDeleteGraphEvidenceByEventIDsZeroInputAvoidsDatabase verifies empty graph evidence deletes return a zero result without touching the database.
func TestDeleteGraphEvidenceByEventIDsZeroInputAvoidsDatabase(t *testing.T) {
	t.Parallel()

	store := NewEntityStore(nil)
	got, err := store.DeleteGraphEvidenceByEventIDs(context.Background(), "workspace", []string{" ", ""})
	if err != nil {
		t.Fatalf("DeleteGraphEvidenceByEventIDs() error = %v", err)
	}
	if !reflect.DeepEqual(got, repository.GraphCleanupResult{}) {
		t.Fatalf("DeleteGraphEvidenceByEventIDs() = %#v, want zero result", got)
	}
}
