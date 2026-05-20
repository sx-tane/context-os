package graph

import (
	"github.com/sx-tane/context-os/shared/entities"
	"github.com/sx-tane/context-os/shared/types"
)

type ContextGraph struct {
	Entities      map[string]entities.CanonicalEntity `json:"entities"`
	Relationships map[string]types.Relationship       `json:"relationships"`
}

func New() *ContextGraph {
	return &ContextGraph{
		Entities:      map[string]entities.CanonicalEntity{},
		Relationships: map[string]types.Relationship{},
	}
}

func (g *ContextGraph) AddEntities(input []entities.CanonicalEntity) {
	for _, entity := range input {
		g.Entities[entity.Entity.ID] = entity
	}
}

func (g *ContextGraph) AddRelationships(input []types.Relationship) {
	for _, relationship := range input {
		g.Relationships[relationship.ID] = relationship
	}
}
