package identity

import (
	"math"    // cosine similarity math
	"strings" // lowercasing names before trigram comparison

	"context-os/domain/entities" // CanonicalEntity output type
	"context-os/domain/types"    // Entity input type
)

// defaultSemanticThreshold is the minimum similarity required to record a semantic
// merge candidate. It is deliberately high so the deterministic exact and convention
// layers remain the primary linkers and semantic matching only surfaces strong,
// review-worthy candidates.
const defaultSemanticThreshold = 0.82

// Matcher scores whether two entity names refer to the same concept. Implementations
// must be deterministic for a given input so pipeline and harness runs stay
// reproducible. The semantic layer uses a Matcher only to propose review candidates;
// it never merges distinct canonical entities automatically.
type Matcher interface {
	// Similarity returns a score in [0,1] estimating that a and b name the same concept.
	Similarity(a, b string) float64
}

// Embedder produces a vector embedding for each input text. The aiworker service
// client satisfies this interface; identity depends only on this narrow contract so
// it stays independent of any concrete service implementation.
type Embedder interface {
	// Embed returns one vector per input text, preserving order.
	Embed(texts []string) ([][]float64, error)
}

// MatchOptions tunes the semantic candidate pass run by ResolveWithMatcher.
type MatchOptions struct {
	// Threshold is the minimum similarity to record a semantic candidate.
	// A zero value falls back to defaultSemanticThreshold.
	Threshold float64
}

// threshold returns the effective similarity threshold for these options.
func (o MatchOptions) threshold() float64 {
	if o.Threshold <= 0 {
		return defaultSemanticThreshold
	}
	return o.Threshold
}

// LocalMatcher is the default, network-free Matcher. It scores similarity using a
// deterministic character-trigram cosine, so identical inputs always yield identical
// scores without contacting any external service.
type LocalMatcher struct{}

// Similarity scores a and b using lowercase character-trigram cosine similarity.
func (LocalMatcher) Similarity(a, b string) float64 {
	return trigramCosine(strings.ToLower(a), strings.ToLower(b))
}

// WorkerMatcher is an opt-in Matcher backed by an external embedding service. It is
// never used by default; callers must pass it explicitly to ResolveWithMatcher. On
// any embedding error it fails closed by returning a zero score, never panicking the
// pipeline.
type WorkerMatcher struct {
	Embedder Embedder // service client used to embed entity names
}

// Similarity embeds both names and returns their cosine similarity, or 0 on error.
func (m WorkerMatcher) Similarity(a, b string) float64 {
	if m.Embedder == nil {
		return 0
	}
	vecs, err := m.Embedder.Embed([]string{a, b})
	if err != nil || len(vecs) != 2 {
		return 0 // fail closed: no semantic signal on error
	}
	return cosineVectors(vecs[0], vecs[1])
}

// ResolveWithMatcher runs the deterministic Resolve layers, then uses matcher to
// propose semantic merge candidates between distinct canonical entities. Candidates
// are recorded on both entities and flag them for human review; ResolveWithMatcher
// never merges distinct canonical entities, so downstream stages stay deterministic.
// A nil matcher defaults to LocalMatcher.
func ResolveWithMatcher(input []types.Entity, matcher Matcher, opts MatchOptions) []entities.CanonicalEntity {
	resolved := Resolve(input)
	if matcher == nil {
		matcher = LocalMatcher{}
	}
	threshold := opts.threshold()

	for i := range resolved { // compare every distinct canonical pair once, in deterministic order
		for j := i + 1; j < len(resolved); j++ {
			score := matcher.Similarity(resolved[i].Entity.Name, resolved[j].Entity.Name)
			if score < threshold {
				continue
			}
			addSemanticCandidate(&resolved[i], resolved[j].Entity.Name, score)
			addSemanticCandidate(&resolved[j], resolved[i].Entity.Name, score)
		}
	}
	return resolved
}

// addSemanticCandidate records a proposed semantic alias on a canonical entity and
// flags it for human confirmation without merging the two concepts.
func addSemanticCandidate(entity *entities.CanonicalEntity, alias string, score float64) {
	entity.Candidates = append(entity.Candidates, entities.MergeCandidate{
		Alias:      alias,
		Layer:      entities.MatchLayerSemantic,
		Confidence: score,
		Evidence:   "semantic similarity above threshold",
		Accepted:   false,
	})
	entity.NeedsHuman = true
	entity.Evidence = append(entity.Evidence, "semantic: proposed candidate "+alias+" needs review")
	if entity.ConflictReason == "" {
		entity.ConflictReason = "semantic candidate(s) need review"
	}
}

// trigramCosine computes cosine similarity between the character-trigram frequency
// vectors of a and b. Short strings are padded so single-token names still produce a
// stable, non-degenerate signal.
func trigramCosine(a, b string) float64 {
	va := trigrams(a)
	vb := trigrams(b)
	var dot, na, nb float64
	for gram, count := range va {
		na += float64(count * count)
		if other, ok := vb[gram]; ok {
			dot += float64(count * other)
		}
	}
	for _, count := range vb {
		nb += float64(count * count)
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// trigrams builds a character-trigram frequency map for a padded string.
func trigrams(value string) map[string]int {
	padded := "  " + value + "  "
	runes := []rune(padded)
	out := map[string]int{}
	for i := 0; i+3 <= len(runes); i++ {
		out[string(runes[i:i+3])]++
	}
	return out
}

// cosineVectors computes cosine similarity between two equal-length float vectors,
// returning 0 when lengths differ or either vector has zero magnitude.
func cosineVectors(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
