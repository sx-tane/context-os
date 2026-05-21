package graph

import (
	"github.com/sx-tane/context-os/domain/entities" // CanonicalEntity stored in the graph
	"github.com/sx-tane/context-os/domain/types"    // Relationship stored in the graph
)

// ContextGraph is the in-memory store for all resolved entities and relationships.
type ContextGraph struct {
	Entities      map[string]entities.CanonicalEntity `json:"entities"`      // keyed by entity ID for O(1) lookup
	Relationships map[string]types.Relationship       `json:"relationships"` // keyed by relationship ID for O(1) lookup
}

// New returns an empty ContextGraph ready to receive entities and relationships.
func New() *ContextGraph {
	return &ContextGraph{
		Entities:      map[string]entities.CanonicalEntity{}, // initialise to avoid nil map panics on write
		Relationships: map[string]types.Relationship{},       // initialise to avoid nil map panics on write
	}
}

// AddEntities inserts or overwrites each entity in the graph by its ID.
func (g *ContextGraph) AddEntities(input []entities.CanonicalEntity) {
	for _, entity := range input {
		g.Entities[entity.Entity.ID] = entity // use the entity ID as key so duplicates overwrite cleanly
	}
}

// AddRelationships inserts or overwrites each relationship in the graph by its ID.
func (g *ContextGraph) AddRelationships(input []types.Relationship) {
	for _, relationship := range input {
		g.Relationships[relationship.ID] = relationship // use the relationship ID as key so duplicates overwrite cleanly
	}
}
