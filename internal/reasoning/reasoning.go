package reasoning

import (
	"fmt"
	"strings"

	"github.com/sx-tane/context-os/internal/graph"
	"github.com/sx-tane/context-os/domain/types"
)

func DetectMismatches(g *graph.ContextGraph) []types.Mismatch {
	mismatches := []types.Mismatch{}
	for _, entity := range g.Entities {
		name := strings.ToLower(entity.Entity.Name)
		if strings.Contains(name, "missing") || strings.Contains(name, "mismatch") || strings.Contains(name, "outdated") {
			mismatches = append(mismatches, types.Mismatch{
				ID:          fmt.Sprintf("mismatch:%s", entity.Entity.ID),
				Summary:     fmt.Sprintf("Potential delivery mismatch around %s", entity.Entity.Name),
				EntityIDs:   []string{entity.Entity.ID},
				Severity:    "medium",
				Recommended: "Confirm FE and BE understanding against the canonical context graph.",
			})
		}
	}
	return mismatches
}
