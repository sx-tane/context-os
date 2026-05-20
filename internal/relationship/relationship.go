package relationship

import (
	"fmt"

	"github.com/sx-tane/context-os/shared/entities"
	"github.com/sx-tane/context-os/shared/types"
)

func Build(canonical []entities.CanonicalEntity) []types.Relationship {
	relationships := []types.Relationship{}
	for i := 0; i < len(canonical)-1; i++ {
		from := canonical[i].Entity
		to := canonical[i+1].Entity
		if from.SourceID != to.SourceID {
			continue
		}
		relationships = append(relationships, types.Relationship{
			ID:       fmt.Sprintf("%s->%s", from.ID, to.ID),
			FromID:   from.ID,
			ToID:     to.ID,
			Kind:     "co_occurs_in_document",
			Metadata: map[string]string{"source_id": from.SourceID},
		})
	}
	return relationships
}
