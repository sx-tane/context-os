// Package pipelines defines the domain-level contract and result type for a full pipeline run.
// The orchestration logic that calls internal stages lives in internal/pipeline.
package pipelines

import (
	"context-os/domain/entities" // CanonicalEntity accumulated during the run
	"context-os/domain/types"    // Mismatch detected by the reasoning stage
)

// Result is the output produced by a full pipeline run.
type Result struct {
	Entities      []entities.CanonicalEntity `json:"entities"`      // all canonical entities accumulated during the run
	Relationships []types.Relationship       `json:"relationships"` // all typed relationships built from co-occurrence and classification
	Mismatches    []types.Mismatch           `json:"mismatches"`    // delivery misalignments detected by the reasoning stage
}
