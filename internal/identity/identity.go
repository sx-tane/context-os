package identity

import (
	"regexp"
	"strings"

	"github.com/sx-tane/context-os/shared/entities"
	"github.com/sx-tane/context-os/shared/types"
)

var separatorPattern = regexp.MustCompile(`[^a-z0-9]+`)

func Resolve(input []types.Entity) []entities.CanonicalEntity {
	canonical := map[string]entities.CanonicalEntity{}
	order := []string{}
	for _, entity := range input {
		key := CanonicalKey(entity.Name)
		current, exists := canonical[key]
		if !exists {
			entity.Aliases = append(entity.Aliases, entity.Name)
			canonical[key] = entities.CanonicalEntity{Entity: entity, Confidence: 1, NeedsHuman: false}
			order = append(order, key)
			continue
		}
		current.Entity.Aliases = appendUnique(current.Entity.Aliases, entity.Name)
		canonical[key] = current
	}
	out := make([]entities.CanonicalEntity, 0, len(order))
	for _, key := range order {
		out = append(out, canonical[key])
	}
	return out
}

func CanonicalKey(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	return separatorPattern.ReplaceAllString(lower, "")
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
