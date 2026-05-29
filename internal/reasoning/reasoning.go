package reasoning

import (
	"fmt"     // used to build deterministic mismatch IDs
	"strings" // used to lowercase entity names for keyword matching

	"context-os/domain/contracts" // source URI metadata used for evidence references
	"context-os/domain/entities"  // CanonicalEntity scanned for mismatch signals
	"context-os/domain/types"     // Mismatch output type
)

// EntityReader provides read access to the entities accumulated in a context graph.
// Accepting this interface instead of the concrete ContextGraph keeps the reasoning
// stage independent of the graph stage.
type EntityReader interface {
	AllEntities() []entities.CanonicalEntity
}

// DetectMismatches scans every entity in g for keyword signals that suggest a delivery misalignment.
func DetectMismatches(g EntityReader) []types.Mismatch {
	mismatches := []types.Mismatch{}         // start with empty results; most entities will not match
	for _, entity := range g.AllEntities() { // inspect every entity that was accumulated in this run
		name := strings.ToLower(entity.Entity.Name) // lowercase the name so keyword checks are case-insensitive
		if strings.Contains(name, "missing") || strings.Contains(name, "mismatch") || strings.Contains(name, "outdated") {
			mismatches = append(mismatches, types.Mismatch{
				ID:          fmt.Sprintf("mismatch:%s", entity.Entity.ID),                                                               // stable ID derived from the entity that triggered the finding
				Type:        "keyword_signal",                                                                                           // deterministic rule category for auditing
				Summary:     fmt.Sprintf("Potential delivery mismatch around %s", entity.Entity.Name),                                   // human-readable description of what was flagged
				EntityIDs:   []string{entity.Entity.ID},                                                                                 // reference back to the triggering entity
				Severity:    "medium",                                                                                                   // keyword-only detection is treated as medium confidence by default
				Confidence:  0.70,                                                                                                       // deterministic confidence for this explainable keyword rule
				Impact:      "medium",                                                                                                   // keyword-only findings should be reviewed but are not yet blockers
				Evidence:    entityEvidence(entity),                                                                                     // cite the source artifact that produced the entity
				Recommended: "Confirm cross-layer understanding across all knowledge participants against the canonical context graph.", // suggested next step for the team
			})
		}
	}
	return mismatches
}

func entityEvidence(entity entities.CanonicalEntity) []string {
	source := strings.TrimSpace(entity.Entity.Metadata[contracts.MetadataSourceURI])
	if source == "" {
		source = strings.TrimSpace(entity.Entity.SourceID)
	}
	if source == "" {
		source = strings.TrimSpace(entity.Entity.ID)
	}
	name := strings.TrimSpace(entity.Entity.Name)
	if name == "" {
		return []string{source}
	}
	return []string{fmt.Sprintf("%s#%s", source, name)}
}
