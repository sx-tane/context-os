package entities

import "context-os/domain/types" // provides the core Entity type

// Match layer constants describe which resolution strategy linked an alias to a canonical entity.
const (
	MatchLayerExact        = "exact"        // identical after canonical-key normalization
	MatchLayerConvention   = "convention"   // matched via naming-convention transforms (snake/camel/kebab/Pascal)
	MatchLayerSemantic     = "semantic"     // matched via embedding similarity above threshold
	MatchLayerRelationship = "relationship" // reserved: matched via shared relationships
	MatchLayerHistorical   = "historical"   // reserved: matched via prior merge memory
)

// MergeCandidate records a proposed alias link that was considered during resolution.
type MergeCandidate struct {
	Alias      string  `json:"alias"`      // the candidate name that may refer to the same concept
	Layer      string  `json:"layer"`      // which match layer proposed this candidate
	Confidence float64 `json:"confidence"` // 0.0–1.0 score for the proposed link
	Evidence   string  `json:"evidence"`   // short human-readable justification for the candidate
	Accepted   bool    `json:"accepted"`   // true when the candidate was merged into the canonical entity
}

// CanonicalEntity wraps a resolved entity with its confidence score and review flag.
type CanonicalEntity struct {
	Entity         types.Entity     `json:"entity"`          // the deduplicated entity with all merged aliases
	Confidence     float64          `json:"confidence"`      // 0.0–1.0 score reflecting how certain the resolution is
	NeedsHuman     bool             `json:"needs_human"`     // true when confidence is too low for automatic resolution
	MatchLayer     string           `json:"match_layer"`     // highest-precedence layer that contributed to this entity
	ConflictReason string           `json:"conflict_reason"` // why human review is needed, when NeedsHuman is set
	Evidence       []string         `json:"evidence"`        // provenance notes explaining how aliases were merged
	Candidates     []MergeCandidate `json:"candidates"`      // proposed alias links considered during resolution
}
