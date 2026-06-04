package graph_test

import (
	"os"
	"path/filepath"
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/stages/graph"
)

// TestSaveAndLoadSnapshotRoundTrips verifies a persisted graph reloads with entities, relationships, and history intact.
func TestSaveAndLoadSnapshotRoundTrips(t *testing.T) {
	original := graph.New()
	original.AddEntities([]entities.CanonicalEntity{
		{Entity: types.Entity{ID: "entity-1", Name: "old"}},
	})
	original.AddEntities([]entities.CanonicalEntity{
		{Entity: types.Entity{ID: "entity-1", Name: "new"}},
	})
	original.AddRelationships([]types.Relationship{
		{ID: "rel-1", FromID: "entity-1", ToID: "entity-2", Kind: "requirement_affects_api"},
	})

	dir := t.TempDir()
	path, err := original.SaveSnapshot(dir, "snapshot")
	if err != nil {
		t.Fatalf("SaveSnapshot() error = %v", err)
	}

	loaded, err := graph.LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot() error = %v", err)
	}
	if got := loaded.Entities["entity-1"].Entity.Name; got != "new" {
		t.Fatalf("loaded Entities[entity-1].Name = %q, want new", got)
	}
	if got := len(loaded.EntityHistory["entity-1"]); got != 2 {
		t.Fatalf("loaded EntityHistory[entity-1] length = %d, want 2", got)
	}
	if got := loaded.Relationships["rel-1"].Kind; got != "requirement_affects_api" {
		t.Fatalf("loaded Relationships[rel-1].Kind = %q, want requirement_affects_api", got)
	}
}

// TestSaveSnapshotIsDeterministic verifies the same graph state always produces byte-identical output for replay safety.
func TestSaveSnapshotIsDeterministic(t *testing.T) {
	build := func() *graph.ContextGraph {
		g := graph.New()
		g.AddEntities([]entities.CanonicalEntity{
			{Entity: types.Entity{ID: "entity-2", Name: "b"}},
			{Entity: types.Entity{ID: "entity-1", Name: "a"}},
		})
		g.AddRelationships([]types.Relationship{
			{ID: "rel-2", Kind: "api_backed_by_db"},
			{ID: "rel-1", Kind: "requirement_affects_api"},
		})
		return g
	}

	dir := t.TempDir()
	firstPath, err := build().SaveSnapshot(dir, "first")
	if err != nil {
		t.Fatalf("SaveSnapshot() first error = %v", err)
	}
	secondPath, err := build().SaveSnapshot(dir, "second")
	if err != nil {
		t.Fatalf("SaveSnapshot() second error = %v", err)
	}

	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("snapshot output is not deterministic:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestLoadSnapshotRejectsUnknownSchema verifies replay refuses snapshot formats it cannot safely decode.
func TestLoadSnapshotRejectsUnknownSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"v999"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := graph.LoadSnapshot(path); err == nil {
		t.Fatalf("LoadSnapshot() error = nil, want unsupported schema error")
	}
}
