package extraction

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sx-tane/context-os/shared/types"
)

var tokenPattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*(?:Status|State|ID|Id|Type|Flag|Field|Column)?`)

func Extract(doc types.ClassifiedDocument) []types.Entity {
	matches := tokenPattern.FindAllString(doc.Document.Body, -1)
	seen := map[string]bool{}
	entities := make([]types.Entity, 0, len(matches))
	for _, match := range matches {
		name := strings.TrimSpace(match)
		key := strings.ToLower(name)
		if len(name) < 3 || seen[key] {
			continue
		}
		seen[key] = true
		entities = append(entities, types.Entity{
			ID:       fmt.Sprintf("%s:%s", doc.Document.ID, key),
			Type:     inferType(name, doc.Classification),
			Name:     name,
			SourceID: doc.Document.ID,
			Metadata: map[string]string{"classification": string(doc.Classification)},
		})
	}
	return entities
}

func inferType(name string, classification types.Classification) types.EntityType {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "field") || strings.Contains(lower, "status") || strings.Contains(lower, "state"):
		return types.APIField
	case strings.Contains(lower, "column") || strings.Contains(lower, "database"):
		return types.DBColumn
	case strings.Contains(lower, "type") || strings.Contains(lower, "flag"):
		return types.Enum
	case classification == types.BusinessLogic:
		return types.Requirement
	case classification == types.APIDiscussion:
		return types.Service
	default:
		return types.Dependency
	}
}
