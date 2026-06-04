package relationship

import (
	"fmt" // used to build deterministic relationship IDs

	"context-os/domain/entities" // CanonicalEntity input type
	"context-os/domain/types"    // Relationship output type
)

const maxCoOccurrenceEdgesPerSource = 25

// Build derives typed relationships between canonical entities that share a source document.
//
// It pairs every distinct entity ordering within the same source and classifies the edge using
// the entity types. Pairs that do not match a typed delivery rule fall back to a co-occurrence
// edge so the context graph still records that the concepts appeared together. Invalid edges are
// dropped via Validate so they never reach persistent storage.
func Build(canonical []entities.CanonicalEntity) []types.Relationship {
	relationships := []types.Relationship{} // start empty; not every pair qualifies as a valid edge
	seen := map[string]struct{}{}           // dedupe by relationship ID so repeated pairs collapse cleanly
	coOccurrencesBySource := map[string]int{}
	for i := 0; i < len(canonical); i++ {
		from := canonical[i].Entity
		for j := i + 1; j < len(canonical); j++ {
			to := canonical[j].Entity
			if from.SourceID != to.SourceID { // only link entities discovered in the same document
				continue
			}
			rel := classify(from, to) // orient and type the edge from the two entities
			if err := Validate(rel); err != nil {
				continue // never emit structurally invalid edges
			}
			if rel.Kind == types.CoOccursInDocument {
				sourceID := rel.Metadata["source_id"]
				if coOccurrencesBySource[sourceID] >= maxCoOccurrenceEdgesPerSource {
					continue
				}
				coOccurrencesBySource[sourceID]++
			}
			if _, dup := seen[rel.ID]; dup {
				continue
			}
			seen[rel.ID] = struct{}{}
			relationships = append(relationships, rel)
		}
	}
	return relationships
}

// Validate reports why a relationship is structurally invalid, or nil when it is safe to persist.
func Validate(rel types.Relationship) error {
	switch {
	case rel.FromID == "":
		return fmt.Errorf("relationship %q has empty from_id", rel.ID)
	case rel.ToID == "":
		return fmt.Errorf("relationship %q has empty to_id", rel.ID)
	case rel.FromID == rel.ToID:
		return fmt.Errorf("relationship %q is a self-loop on %q", rel.ID, rel.FromID)
	case rel.Kind == "":
		return fmt.Errorf("relationship %q has empty kind", rel.ID)
	default:
		return nil
	}
}

// classify orients a pair of same-source entities into a typed, directed relationship.
// When no typed delivery rule applies it falls back to an auditable co-occurrence edge.
func classify(a, b types.Entity) types.Relationship {
	if kind, from, to, ok := typedKind(a, b); ok {
		return edge(from, to, kind, 0.8)
	}
	if isLowConfidenceRegexOnly(a, b) {
		return types.Relationship{}
	}
	return edge(a, b, types.CoOccursInDocument, 0.5)
}

// typedKind returns the delivery-semantic edge for a pair, oriented from->to, when one applies.
func typedKind(a, b types.Entity) (types.RelationshipKind, types.Entity, types.Entity, bool) {
	pairs := []struct {
		from types.EntityType
		to   types.EntityType
		kind types.RelationshipKind
	}{
		{types.Requirement, types.APIField, types.RequirementAffectsAPI},
		{types.Requirement, types.Service, types.RequirementAffectsService},
		{types.APIField, types.DBColumn, types.APIBackedByDB},
		{types.Enum, types.APIField, types.EnumConstrainsField},
		{types.Enum, types.DBColumn, types.EnumConstrainsField},
		{types.Service, types.Dependency, types.ServiceDependsOn},
	}
	for _, p := range pairs {
		if a.Type == p.from && b.Type == p.to {
			return p.kind, a, b, true
		}
		if b.Type == p.from && a.Type == p.to { // same semantic edge, reversed input order
			return p.kind, b, a, true
		}
	}
	return "", types.Entity{}, types.Entity{}, false
}

// edge builds a deterministic, evidence-backed relationship between two oriented entities.
func edge(from, to types.Entity, kind types.RelationshipKind, confidence float64) types.Relationship {
	return types.Relationship{
		ID:         relationshipID(from.ID, to.ID, kind), // ID includes kind so distinct edge types never collide
		FromID:     from.ID,
		ToID:       to.ID,
		Kind:       kind,
		Confidence: confidence,
		Evidence: []string{ // cite both endpoints so reasoning can trace the edge to source entities
			fmt.Sprintf("%s#%s", from.SourceID, from.Name),
			fmt.Sprintf("%s#%s", to.SourceID, to.Name),
		},
		Metadata: map[string]string{"source_id": from.SourceID}, // record which document produced this edge
	}
}

func relationshipID(fromID, toID string, kind types.RelationshipKind) string {
	return fmt.Sprintf("%s->%s:%s", fromID, toID, kind)
}

func isLowConfidenceRegexOnly(a, b types.Entity) bool {
	if a.ExtractionMethod != "regex_token" || b.ExtractionMethod != "regex_token" {
		return false
	}
	return a.Confidence < 0.6 || b.Confidence < 0.6
}
