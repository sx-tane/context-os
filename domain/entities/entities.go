package entities

import "context-os/domain/types" // provides the core Entity type

// CanonicalEntity wraps a resolved entity with its confidence score and review flag.
type CanonicalEntity struct {
	Entity     types.Entity `json:"entity"`      // the deduplicated entity with all merged aliases
	Confidence float64      `json:"confidence"`  // 0.0–1.0 score reflecting how certain the resolution is
	NeedsHuman bool         `json:"needs_human"` // true when confidence is too low for automatic resolution
}
