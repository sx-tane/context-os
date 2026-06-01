package graph

import (
	"context-os/domain/entities" // CanonicalEntity stored in the graph
	"context-os/domain/types"    // Relationship stored in the graph
)

// ContextGraph is the in-memory store for all resolved entities and relationships.
// It preserves prior versions of every entity and relationship so graph history can be
// inspected and replayed rather than silently overwritten.
type ContextGraph struct {
	Entities            map[string]entities.CanonicalEntity   `json:"entities"`             // current entity by ID for O(1) lookup
	Relationships       map[string]types.Relationship         `json:"relationships"`        // current relationship by ID for O(1) lookup
	EntityHistory       map[string][]entities.CanonicalEntity `json:"entity_history"`       // every recorded version of an entity by ID
	RelationshipHistory map[string][]types.Relationship       `json:"relationship_history"` // every recorded version of a relationship by ID
}

// New returns an empty ContextGraph ready to receive entities and relationships.
func New() *ContextGraph {
	return &ContextGraph{
		Entities:            map[string]entities.CanonicalEntity{},   // initialise to avoid nil map panics on write
		Relationships:       map[string]types.Relationship{},         // initialise to avoid nil map panics on write
		EntityHistory:       map[string][]entities.CanonicalEntity{}, // history is append-only per entity ID
		RelationshipHistory: map[string][]types.Relationship{},       // history is append-only per relationship ID
	}
}

// AddEntities inserts or overwrites each entity by ID and appends every version to history.
func (g *ContextGraph) AddEntities(input []entities.CanonicalEntity) {
	for _, entity := range input {
		id := entity.Entity.ID
		g.Entities[id] = entity                                   // current materialised view overwrites by ID
		g.EntityHistory[id] = append(g.EntityHistory[id], entity) // preserve prior versions for replay and audit
	}
}

// AddRelationships inserts or overwrites each relationship by ID and appends every version to history.
func (g *ContextGraph) AddRelationships(input []types.Relationship) {
	for _, relationship := range input {
		id := relationship.ID
		g.Relationships[id] = relationship                                          // current materialised view overwrites by ID
		g.RelationshipHistory[id] = append(g.RelationshipHistory[id], relationship) // preserve prior versions for replay and audit
	}
}

// AllEntities returns all canonical entities stored in the graph as a slice.
// Order is not guaranteed; callers that need stable output should sort after receiving.
func (g *ContextGraph) AllEntities() []entities.CanonicalEntity {
	out := make([]entities.CanonicalEntity, 0, len(g.Entities))
	for _, e := range g.Entities {
		out = append(out, e)
	}
	return out
}

// AllRelationships returns all relationships stored in the graph as a slice.
// Order is not guaranteed; callers that need stable output should sort after receiving.
func (g *ContextGraph) AllRelationships() []types.Relationship {
	out := make([]types.Relationship, 0, len(g.Relationships))
	for _, r := range g.Relationships {
		out = append(out, r)
	}
	return out
}

// Neighbors returns the relationships directly incident to the given entity ID
// (either as the source or the target of the edge).
func (g *ContextGraph) Neighbors(entityID string) []types.Relationship {
	out := []types.Relationship{}
	for _, r := range g.Relationships {
		if r.FromID == entityID || r.ToID == entityID {
			out = append(out, r)
		}
	}
	return out
}

// ImpactOf returns every entity ID reachable from the given entity ID by following
// directed edges outward. It supports impact analysis across requirement, API, DB,
// service, and dependency artifacts. The starting entity is not included in the result.
func (g *ContextGraph) ImpactOf(entityID string) []string {
	visited := map[string]bool{entityID: true}
	queue := []string{entityID}
	reached := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, r := range g.Relationships {
			if r.FromID != current || visited[r.ToID] {
				continue
			}
			visited[r.ToID] = true
			reached = append(reached, r.ToID)
			queue = append(queue, r.ToID)
		}
	}
	return reached
}
