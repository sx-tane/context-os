package entities

import "github.com/sx-tane/context-os/domain/types"

type CanonicalEntity struct {
	Entity     types.Entity `json:"entity"`
	Confidence float64      `json:"confidence"`
	NeedsHuman bool         `json:"needs_human"`
}
