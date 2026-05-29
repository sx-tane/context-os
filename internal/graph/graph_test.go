package graph_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/graph"
)

// TestContextGraphStoresEntitiesAndRelationshipsByID verifies graph inserts overwrite by stable identifiers.
func TestContextGraphStoresEntitiesAndRelationshipsByID(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddEntities([]entities.CanonicalEntity{
		{Entity: types.Entity{ID: "entity-1", Name: "old"}},
		{Entity: types.Entity{ID: "entity-1", Name: "new"}},
	})
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "rel-1", Kind: "old"},
		{ID: "rel-1", Kind: "new"},
	})

	if len(contextGraph.Entities) != 1 {
		t.Fatalf("Entities length = %d, want 1", len(contextGraph.Entities))
	}
	if contextGraph.Entities["entity-1"].Entity.Name != "new" {
		t.Fatalf("Entities[entity-1].Name = %q, want new", contextGraph.Entities["entity-1"].Entity.Name)
	}
	if len(contextGraph.Relationships) != 1 {
		t.Fatalf("Relationships length = %d, want 1", len(contextGraph.Relationships))
	}
	if contextGraph.Relationships["rel-1"].Kind != "new" {
		t.Fatalf("Relationships[rel-1].Kind = %q, want new", contextGraph.Relationships["rel-1"].Kind)
	}
	if len(contextGraph.AllEntities()) != 1 {
		t.Fatalf("AllEntities() length = %d, want 1", len(contextGraph.AllEntities()))
	}
}
