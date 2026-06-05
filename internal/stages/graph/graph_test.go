package graph_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/stages/graph"
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

// TestContextGraphShortestPathFindsCheapestDirectedPath verifies multi-hop high-confidence edges beat a low-confidence direct edge.
func TestContextGraphShortestPathFindsCheapestDirectedPath(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "a->d", FromID: "a", ToID: "d", Confidence: 0.2},
		{ID: "a->b", FromID: "a", ToID: "b", Confidence: 1},
		{ID: "b->d", FromID: "b", ToID: "d", Confidence: 1},
	})

	got := contextGraph.ShortestPath("a", "d")
	if !got.Found {
		t.Fatalf("ShortestPath(a, d).Found = false, want true")
	}
	if got.TotalCost != 2 {
		t.Fatalf("ShortestPath(a, d).TotalCost = %v, want 2", got.TotalCost)
	}
	assertPathSteps(t, got.Steps, []graph.PathStep{
		{EntityID: "a"},
		{EntityID: "b", RelationshipID: "a->b", Cost: 1},
		{EntityID: "d", RelationshipID: "b->d", Cost: 1},
	})
}

// TestContextGraphShortestPathReturnsNoPathForUnreachableTarget verifies unreachable targets produce an empty result.
func TestContextGraphShortestPathReturnsNoPathForUnreachableTarget(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "a->b", FromID: "a", ToID: "b", Confidence: 1},
		{ID: "c->d", FromID: "c", ToID: "d", Confidence: 1},
	})

	got := contextGraph.ShortestPath("a", "d")
	if got.Found {
		t.Fatalf("ShortestPath(a, d).Found = true, want false")
	}
	if got.TotalCost != 0 {
		t.Fatalf("ShortestPath(a, d).TotalCost = %v, want 0", got.TotalCost)
	}
	if len(got.Steps) != 0 {
		t.Fatalf("ShortestPath(a, d).Steps length = %d, want 0", len(got.Steps))
	}
}

// TestContextGraphShortestPathDoesNotFollowReverseEdges verifies traversal only follows relationships from source to target.
func TestContextGraphShortestPathDoesNotFollowReverseEdges(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "b->a", FromID: "b", ToID: "a", Confidence: 1},
	})

	got := contextGraph.ShortestPath("a", "b")
	if got.Found {
		t.Fatalf("ShortestPath(a, b).Found = true, want false")
	}
	if len(got.Steps) != 0 {
		t.Fatalf("ShortestPath(a, b).Steps length = %d, want 0", len(got.Steps))
	}
}

// TestContextGraphShortestPathReturnsSelfPath verifies identical endpoints return a zero-cost entity step.
func TestContextGraphShortestPathReturnsSelfPath(t *testing.T) {
	contextGraph := graph.New()

	got := contextGraph.ShortestPath("a", "a")
	if !got.Found {
		t.Fatalf("ShortestPath(a, a).Found = false, want true")
	}
	if got.TotalCost != 0 {
		t.Fatalf("ShortestPath(a, a).TotalCost = %v, want 0", got.TotalCost)
	}
	assertPathSteps(t, got.Steps, []graph.PathStep{{EntityID: "a"}})
}

// TestContextGraphShortestPathChoosesDeterministicEqualCostPath verifies ties prefer the lexicographically smallest path.
func TestContextGraphShortestPathChoosesDeterministicEqualCostPath(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "a->c", FromID: "a", ToID: "c", Confidence: 1},
		{ID: "a->b", FromID: "a", ToID: "b", Confidence: 1},
		{ID: "c->d", FromID: "c", ToID: "d", Confidence: 1},
		{ID: "b->d", FromID: "b", ToID: "d", Confidence: 1},
	})

	got := contextGraph.ShortestPath("a", "d")
	if !got.Found {
		t.Fatalf("ShortestPath(a, d).Found = false, want true")
	}
	if got.TotalCost != 2 {
		t.Fatalf("ShortestPath(a, d).TotalCost = %v, want 2", got.TotalCost)
	}
	assertPathSteps(t, got.Steps, []graph.PathStep{
		{EntityID: "a"},
		{EntityID: "b", RelationshipID: "a->b", Cost: 1},
		{EntityID: "d", RelationshipID: "b->d", Cost: 1},
	})
}

// TestContextGraphShortestPathUsesDefaultCostForMissingConfidence verifies zero-confidence edges remain traversable.
func TestContextGraphShortestPathUsesDefaultCostForMissingConfidence(t *testing.T) {
	contextGraph := graph.New()
	contextGraph.AddRelationships([]types.Relationship{
		{ID: "a->b", FromID: "a", ToID: "b"},
	})

	got := contextGraph.ShortestPath("a", "b")
	if !got.Found {
		t.Fatalf("ShortestPath(a, b).Found = false, want true")
	}
	if got.TotalCost != 1 {
		t.Fatalf("ShortestPath(a, b).TotalCost = %v, want 1", got.TotalCost)
	}
	assertPathSteps(t, got.Steps, []graph.PathStep{
		{EntityID: "a"},
		{EntityID: "b", RelationshipID: "a->b", Cost: 1},
	})
}

// assertPathSteps verifies a path has the expected entity and relationship sequence.
func assertPathSteps(t *testing.T, got, want []graph.PathStep) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Steps length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Steps[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
