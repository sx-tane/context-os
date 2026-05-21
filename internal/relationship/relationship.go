package relationship

import (
	"fmt" // used to build deterministic relationship IDs

	"github.com/sx-tane/context-os/domain/entities" // CanonicalEntity input type
	"github.com/sx-tane/context-os/domain/types"    // Relationship output type
)

// Build creates co-occurrence relationships between adjacent canonical entities from the same document.
func Build(canonical []entities.CanonicalEntity) []types.Relationship {
	relationships := []types.Relationship{}         // start with an empty list; not every pair will qualify
	for i := 0; i < len(canonical)-1; i++ {         // iterate adjacent pairs (i, i+1) across the entity list
		from := canonical[i].Entity                  // the left entity in this adjacent pair
		to := canonical[i+1].Entity                  // the right entity in this adjacent pair
		if from.SourceID != to.SourceID {            // only link entities that come from the same document
			continue
		}
		relationships = append(relationships, types.Relationship{
			ID:       fmt.Sprintf("%s->%s", from.ID, to.ID),                 // deterministic ID from the two entity IDs
			FromID:   from.ID,                                               // source entity in the directed edge
			ToID:     to.ID,                                                 // target entity in the directed edge
			Kind:     "co_occurs_in_document",                               // relationship type: both appeared in the same document
			Metadata: map[string]string{"source_id": from.SourceID},         // record which document produced this edge
		})
	}
	return relationships
}
