package reasoning

import (
	"fmt"     // used to build deterministic mismatch IDs
	"strings" // used to lowercase entity names for keyword matching

	"github.com/sx-tane/context-os/domain/types"   // Mismatch output type
	"github.com/sx-tane/context-os/internal/graph" // ContextGraph scanned for mismatches
)

// DetectMismatches scans every entity in the graph for keyword signals that suggest a delivery misalignment.
func DetectMismatches(g *graph.ContextGraph) []types.Mismatch {
	mismatches := []types.Mismatch{}              // start with empty results; most entities will not match
	for _, entity := range g.Entities {           // inspect every entity that was accumulated in this run
		name := strings.ToLower(entity.Entity.Name) // lowercase the name so keyword checks are case-insensitive
		if strings.Contains(name, "missing") || strings.Contains(name, "mismatch") || strings.Contains(name, "outdated") {
			mismatches = append(mismatches, types.Mismatch{
				ID:          fmt.Sprintf("mismatch:%s", entity.Entity.ID),                              // stable ID derived from the entity that triggered the finding
				Summary:     fmt.Sprintf("Potential delivery mismatch around %s", entity.Entity.Name),  // human-readable description of what was flagged
				EntityIDs:   []string{entity.Entity.ID},                                                // reference back to the triggering entity
				Severity:    "medium",                                                                  // keyword-only detection is treated as medium confidence by default
				Recommended: "Confirm FE and BE understanding against the canonical context graph.",    // suggested next step for the team
			})
		}
	}
	return mismatches
}
