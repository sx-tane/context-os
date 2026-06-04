// Package identity resolves extracted entities into canonical entities by merging
// equivalent name variants. Resolution runs in deterministic layers: exact-key
// matching and naming-convention matching collapse aliases automatically, while
// semantic matching only proposes review candidates and never merges distinct
// concepts. This keeps the default pipeline hermetic, explainable, and replay-safe.
package identity

import (
	"fmt"     // formats provenance evidence strings
	"regexp"  // used to strip separators when computing a canonical key
	"sort"    // produces deterministic ordering for conflict evidence
	"strings" // used for lowercasing and trimming entity names

	"context-os/domain/entities" // CanonicalEntity output type
	"context-os/domain/types"    // Entity input type
)

// separatorPattern matches any character that is not a lowercase letter or digit.
var separatorPattern = regexp.MustCompile(`[^a-z0-9]+`)

// conflictConfidence is the resolution confidence assigned when merged aliases
// disagree on entity type, signalling that a human should confirm the merge.
const conflictConfidence = 0.5

// Resolve merges entities that share the same canonical key into a single
// CanonicalEntity using the deterministic exact and convention layers only.
// It preserves source IDs and insertion order, populates provenance evidence,
// and flags type conflicts for human review. Semantic candidate detection is
// available through ResolveWithMatcher.
func Resolve(input []types.Entity) []entities.CanonicalEntity {
	buckets := map[string]*mergeBucket{} // keyed accumulator for deduplication during resolution
	order := []string{}                  // preserves insertion order so output is deterministic

	for _, entity := range input { // process each extracted entity in document order
		key := CanonicalKey(entity.Name) // compute the normalised key for this entity name
		bucket, exists := buckets[key]   // check if this key has been seen before
		if !exists {                     // first time seeing this key — register a new bucket
			buckets[key] = newBucket(key, entity)
			order = append(order, key)
			continue
		}
		bucket.add(entity) // merge the new variant into the existing bucket
	}

	out := make([]entities.CanonicalEntity, 0, len(order)) // pre-allocate output in insertion order
	for _, key := range order {
		out = append(out, buckets[key].finalize()) // resolve each bucket in first-seen order
	}
	return out
}

// CanonicalKey reduces a name to lowercase with all separators removed for consistent identity matching.
func CanonicalKey(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))  // lowercase and strip surrounding whitespace
	return separatorPattern.ReplaceAllString(lower, "") // remove underscores, hyphens, spaces, etc.
}

// mergeBucket accumulates every entity that shares one canonical key so the final
// CanonicalEntity can report how its aliases were linked and whether they conflict.
type mergeBucket struct {
	key      string                    // shared canonical key for every entity in this bucket
	entity   types.Entity              // representative entity, seeded from the first occurrence
	surfaces map[string]bool           // distinct surface forms observed, for exact-vs-convention detection
	types    map[types.EntityType]bool // distinct entity types observed, for conflict detection
	evidence []string                  // provenance notes explaining how aliases were merged
}

// newBucket creates a bucket seeded from the first entity observed for a key.
func newBucket(key string, seed types.Entity) *mergeBucket {
	seed.Aliases = appendUnique(seed.Aliases, seed.Name) // seed the alias list with the entity's own name
	return &mergeBucket{
		key:      key,
		entity:   seed,
		surfaces: map[string]bool{seed.Name: true},
		types:    map[types.EntityType]bool{seed.Type: true},
		evidence: []string{fmt.Sprintf("exact: seeded canonical key %q from %s", key, seed.SourceID)},
	}
}

// add merges another entity that shares this bucket's canonical key.
func (b *mergeBucket) add(entity types.Entity) {
	b.entity.Aliases = appendUnique(b.entity.Aliases, entity.Name) // merge the new name variant into the alias list
	b.types[entity.Type] = true                                    // track its type for conflict detection
	if b.surfaces[entity.Name] {                                   // identical surface form — exact contribution
		return
	}
	b.surfaces[entity.Name] = true
	b.evidence = append(b.evidence, fmt.Sprintf( // convention contribution — record the transform link
		"convention: linked %q to canonical key %q from %s", entity.Name, b.key, entity.SourceID))
}

// finalize converts the accumulated bucket into its resolved CanonicalEntity.
func (b *mergeBucket) finalize() entities.CanonicalEntity {
	resolved := entities.CanonicalEntity{
		Entity:     b.entity,
		Confidence: 1,
		MatchLayer: b.matchLayer(),
		Evidence:   b.evidence,
	}
	if len(b.types) > 1 { // merged aliases disagree on type — needs human confirmation
		resolved.NeedsHuman = true
		resolved.Confidence = conflictConfidence
		resolved.ConflictReason = fmt.Sprintf("type conflict among merged aliases: %s", strings.Join(sortedTypes(b.types), ", "))
		resolved.Evidence = append(resolved.Evidence, "conflict: "+resolved.ConflictReason)
	}
	return resolved
}

// matchLayer reports the most informative layer that linked this bucket's aliases.
func (b *mergeBucket) matchLayer() string {
	if len(b.surfaces) > 1 { // more than one surface form means convention transforms were applied
		return entities.MatchLayerConvention
	}
	return entities.MatchLayerExact
}

// sortedTypes returns the distinct entity types in a deterministic order.
func sortedTypes(set map[types.EntityType]bool) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, string(t))
	}
	sort.Strings(out)
	return out
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
