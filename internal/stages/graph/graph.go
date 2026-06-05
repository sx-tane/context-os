package graph

import (
	"container/heap"
	"sort"
	"strings"

	"context-os/domain/entities" // CanonicalEntity stored in the graph
	"context-os/domain/types"    // Relationship stored in the graph
)

// PathStep describes one entity reached while traversing a shortest path.
// RelationshipID and Cost identify the edge used to arrive at EntityID; the
// starting entity has an empty RelationshipID and zero cost.
type PathStep struct {
	EntityID       string  `json:"entity_id"`
	RelationshipID string  `json:"relationship_id"`
	Cost           float64 `json:"cost"`
}

// ShortestPath describes the cheapest directed path between two entities.
type ShortestPath struct {
	Found     bool       `json:"found"`
	TotalCost float64    `json:"total_cost"`
	Steps     []PathStep `json:"steps"`
}

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

// ShortestPath returns the cheapest directed path from fromID to toID using
// relationship confidence as edge cost. Higher confidence edges are cheaper.
func (g *ContextGraph) ShortestPath(fromID, toID string) ShortestPath {
	if fromID == "" || toID == "" {
		return ShortestPath{}
	}
	if fromID == toID {
		return ShortestPath{
			Found: true,
			Steps: []PathStep{{EntityID: fromID}},
		}
	}

	outgoing := g.outgoingRelationships()
	queue := &pathQueue{}
	heap.Push(queue, &pathItem{
		entityID: fromID,
		pathKey:  fromID,
		steps:    []PathStep{{EntityID: fromID}},
	})

	best := map[string]pathBest{
		fromID: {cost: 0, pathKey: fromID},
	}

	for queue.Len() > 0 {
		current := heap.Pop(queue).(*pathItem)
		known := best[current.entityID]
		if current.cost > known.cost || (current.cost == known.cost && current.pathKey != known.pathKey) {
			continue
		}
		if current.entityID == toID {
			return ShortestPath{
				Found:     true,
				TotalCost: current.cost,
				Steps:     current.steps,
			}
		}

		for _, relationship := range outgoing[current.entityID] {
			nextCost := current.cost + relationshipCost(relationship)
			nextPathKey := current.pathKey + "\x00" + relationship.ToID + "\x00" + relationship.ID
			previous, seen := best[relationship.ToID]
			if seen && (nextCost > previous.cost || (nextCost == previous.cost && nextPathKey >= previous.pathKey)) {
				continue
			}

			nextSteps := append([]PathStep{}, current.steps...)
			nextSteps = append(nextSteps, PathStep{
				EntityID:       relationship.ToID,
				RelationshipID: relationship.ID,
				Cost:           relationshipCost(relationship),
			})
			best[relationship.ToID] = pathBest{cost: nextCost, pathKey: nextPathKey}
			heap.Push(queue, &pathItem{
				entityID: relationship.ToID,
				cost:     nextCost,
				pathKey:  nextPathKey,
				steps:    nextSteps,
			})
		}
	}

	return ShortestPath{}
}

// outgoingRelationships indexes directed edges by source entity with stable edge order.
func (g *ContextGraph) outgoingRelationships() map[string][]types.Relationship {
	outgoing := map[string][]types.Relationship{}
	for _, relationship := range g.Relationships {
		outgoing[relationship.FromID] = append(outgoing[relationship.FromID], relationship)
	}
	for fromID := range outgoing {
		sort.Slice(outgoing[fromID], func(i, j int) bool {
			if outgoing[fromID][i].ToID != outgoing[fromID][j].ToID {
				return outgoing[fromID][i].ToID < outgoing[fromID][j].ToID
			}
			return outgoing[fromID][i].ID < outgoing[fromID][j].ID
		})
	}
	return outgoing
}

// relationshipCost returns the traversal cost for a relationship.
func relationshipCost(relationship types.Relationship) float64 {
	if relationship.Confidence > 0 {
		return 1 / relationship.Confidence
	}
	return 1
}

type pathBest struct {
	cost    float64
	pathKey string
}

type pathItem struct {
	entityID string
	cost     float64
	pathKey  string
	steps    []PathStep
	index    int
}

type pathQueue []*pathItem

func (q pathQueue) Len() int {
	return len(q)
}

func (q pathQueue) Less(i, j int) bool {
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	return strings.Compare(q[i].pathKey, q[j].pathKey) < 0
}

func (q pathQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *pathQueue) Push(x any) {
	item := x.(*pathItem)
	item.index = len(*q)
	*q = append(*q, item)
}

func (q *pathQueue) Pop() any {
	old := *q
	item := old[len(old)-1]
	old[len(old)-1] = nil
	item.index = -1
	*q = old[:len(old)-1]
	return item
}
