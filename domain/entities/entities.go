package entities

import "context-os/domain/types" // provides the core Entity type

// Match layer labels describe which resolution strategy linked a name to a canonical entity.
const (
	MatchLayerExact      = "exact"      // string equality after canonical-key normalization
	MatchLayerConvention = "convention" // naming-convention transforms (snake/camel/kebab/Pascal)
	MatchLayerSemantic   = "semantic"   // embedding similarity above the configured threshold
)

// MergeCandidate records a proposed alias link the resolver was not certain enough to auto-apply.
type MergeCandidate struct {
	Name       string   `json:"name"`       // the candidate name being proposed for merge
	Layer      string   `json:"layer"`      // which match layer surfaced the candidate
	Confidence float64  `json:"confidence"` // 0.0–1.0 similarity/score for the candidate
	Evidence   []string `json:"evidence"`   // references explaining why the candidate was suggested
}

// CanonicalEntity wraps a resolved entity with its confidence score and review flag.
type CanonicalEntity struct {
	Entity         types.Entity     `json:"entity"`          // the deduplicated entity with all merged aliases
	Confidence     float64          `json:"confidence"`      // 0.0–1.0 score reflecting how certain the resolution is
	NeedsHuman     bool             `json:"needs_human"`     // true when a merge is ambiguous or high-impact and needs review
	MatchLayer     string           `json:"match_layer"`     // the strongest match layer that contributed to this entity
	Evidence       []string         `json:"evidence"`        // references explaining how aliases were merged
	ConflictReason string           `json:"conflict_reason"` // populated when NeedsHuman is set, explaining the conflict
	Candidates     []MergeCandidate `json:"candidates"`      // unmerged alias candidates awaiting confirmation
}
