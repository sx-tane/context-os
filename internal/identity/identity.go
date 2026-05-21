package identity

import (
	"regexp"  // used to strip separators when computing a canonical key
	"strings" // used for lowercasing and trimming entity names

	"github.com/sx-tane/context-os/domain/entities" // CanonicalEntity output type
	"github.com/sx-tane/context-os/domain/types"    // Entity input type
)

// separatorPattern matches any character that is not a lowercase letter or digit.
var separatorPattern = regexp.MustCompile(`[^a-z0-9]+`)

// Resolve merges entities that share the same canonical key into a single CanonicalEntity.
func Resolve(input []types.Entity) []entities.CanonicalEntity {
	canonical := map[string]entities.CanonicalEntity{} // keyed map for deduplication during resolution
	order := []string{}                                 // preserves insertion order so output is deterministic
	for _, entity := range input {                      // process each extracted entity in document order
		key := CanonicalKey(entity.Name)                // compute the normalised key for this entity name
		current, exists := canonical[key]               // check if this key has been seen before
		if !exists {                                    // first time seeing this key — register a new canonical entry
			entity.Aliases = append(entity.Aliases, entity.Name)                          // seed the alias list with the entity's own name
			canonical[key] = entities.CanonicalEntity{Entity: entity, Confidence: 1, NeedsHuman: false} // store with full confidence since it's an exact first match
			order = append(order, key)                                                     // record insertion order for deterministic output
			continue
		}
		current.Entity.Aliases = appendUnique(current.Entity.Aliases, entity.Name) // merge the new name variant into the alias list
		canonical[key] = current                                                    // write the updated entry back to the map
	}
	out := make([]entities.CanonicalEntity, 0, len(order)) // pre-allocate output in insertion order
	for _, key := range order {
		out = append(out, canonical[key]) // append each resolved entity in the order it was first seen
	}
	return out
}

// CanonicalKey reduces a name to lowercase with all separators removed for consistent identity matching.
func CanonicalKey(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))    // lowercase and strip surrounding whitespace
	return separatorPattern.ReplaceAllString(lower, "")   // remove underscores, hyphens, spaces, etc.
}

// appendUnique adds next to values only if it is not already present.
func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values // already in the list — return unchanged to avoid duplicates
		}
	}
	return append(values, next) // not found — add it
}
