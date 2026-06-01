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

// TestContextGraphPreservesHistoryOnOverwrite verifies prior versions are retained for replay and audit.
func TestContextGraphPreservesHistoryOnOverwrite(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddEntities([]entities.CanonicalEntity{
		{Entity: types.Entity{ID: "entity-1", Name: "old"}},
	})
	contextGraph.AddEntities([]entities.CanonicalEntity{
		{Entity: types.Entity{ID: "entity-1", Name: "new"}},
	})
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "rel-1", Kind: "old"},
	})
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "rel-1", Kind: "new"},
	})

	if got := len(contextGraph.EntityHistory["entity-1"]); got != 2 {
		t.Fatalf("EntityHistory[entity-1] length = %d, want 2", got)
	}
	if got := contextGraph.EntityHistory["entity-1"][0].Entity.Name; got != "old" {
		t.Fatalf("EntityHistory[entity-1][0].Name = %q, want old", got)
	}
	if got := len(contextGraph.RelationshipHistory["rel-1"]); got != 2 {
		t.Fatalf("RelationshipHistory[rel-1] length = %d, want 2", got)
	}
	if got := contextGraph.RelationshipHistory["rel-1"][0].Kind; got != "old" {
		t.Fatalf("RelationshipHistory[rel-1][0].Kind = %q, want old", got)
	}
}

// TestContextGraphAllRelationshipsReturnsCurrentEdges verifies the relationship accessor mirrors AllEntities.
func TestContextGraphAllRelationshipsReturnsCurrentEdges(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "rel-1", Kind: "requirement_affects_api"},
		{ID: "rel-2", Kind: "api_backed_by_db"},
	})

	if got := len(contextGraph.AllRelationships()); got != 2 {
		t.Fatalf("AllRelationships() length = %d, want 2", got)
	}
}

// TestContextGraphNeighborsReturnsIncidentEdges verifies both incoming and outgoing edges are returned.
func TestContextGraphNeighborsReturnsIncidentEdges(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "a->b", FromID: "a", ToID: "b", Kind: "requirement_affects_api"},
		{ID: "c->a", FromID: "c", ToID: "a", Kind: "service_depends_on"},
		{ID: "d->e", FromID: "d", ToID: "e", Kind: "api_backed_by_db"},
	})

	got := contextGraph.Neighbors("a")
	if len(got) != 2 {
		t.Fatalf("Neighbors(a) length = %d, want 2", len(got))
	}
}

// TestContextGraphImpactOfTraversesOutwardEdges verifies impact analysis reaches transitively connected entities.
func TestContextGraphImpactOfTraversesOutwardEdges(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "req->api", FromID: "req", ToID: "api", Kind: "requirement_affects_api"},
		{ID: "api->db", FromID: "api", ToID: "db", Kind: "api_backed_by_db"},
		{ID: "other->x", FromID: "other", ToID: "x", Kind: "co_occurs_in_document"},
	})

	got := contextGraph.ImpactOf("req")
	reached := map[string]bool{}
	for _, id := range got {
		reached[id] = true
	}
	if len(got) != 2 || !reached["api"] || !reached["db"] {
		t.Fatalf("ImpactOf(req) = %v, want [api db]", got)
	}
}
