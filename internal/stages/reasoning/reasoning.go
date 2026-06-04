package reasoning

import (
	"fmt"     // build deterministic mismatch IDs and summaries
	"sort"    // produce deterministic, stable finding ordering
	"strings" // lowercase entity names for keyword matching

	"context-os/domain/contracts" // source URI metadata used for evidence references
	"context-os/domain/entities"  // CanonicalEntity scanned for mismatch signals
	"context-os/domain/types"     // Mismatch output type and relationship vocabulary
)

// GraphReader provides read access to the entities and relationships accumulated in a
// context graph. Accepting this interface instead of the concrete ContextGraph keeps the
// reasoning stage independent of the graph stage.
type GraphReader interface {
	AllEntities() []entities.CanonicalEntity
	AllRelationships() []types.Relationship
}

// DetectMismatches reads the context graph and reports actionable delivery misalignments.
// It runs a deterministic set of rules over entities and typed relationships, emitting
// findings with evidence, confidence, impact, severity, affected roles, and a recommended action.
// Findings are returned sorted by ID so the same graph always produces the same report.
func DetectMismatches(g GraphReader) []types.Mismatch {
	all := g.AllEntities()
	sort.Slice(all, func(i, j int) bool { return all[i].Entity.ID < all[j].Entity.ID })

	rels := g.AllRelationships()
	outgoing := outgoingByEntity(rels) // index edges by source entity for gap and drift checks
	incoming := incomingByEntity(rels) // index edges by target entity for drift context checks

	mismatches := []types.Mismatch{}
	for _, entity := range all {
		if m, ok := keywordSignal(entity); ok {
			mismatches = append(mismatches, m)
		}
		if m, ok := requirementGap(entity, outgoing); ok {
			mismatches = append(mismatches, m)
		}
		if m, ok := crossLayerContractDrift(entity, outgoing, incoming); ok {
			mismatches = append(mismatches, m)
		}
	}

	sortedRels := append([]types.Relationship(nil), rels...)
	sort.Slice(sortedRels, func(i, j int) bool { return sortedRels[i].ID < sortedRels[j].ID })
	for _, rel := range sortedRels {
		if m, ok := dependencyRisk(rel); ok {
			mismatches = append(mismatches, m)
		}
	}

	sort.Slice(mismatches, func(i, j int) bool { return mismatches[i].ID < mismatches[j].ID })
	return mismatches
}

// keywordSignal flags entities whose name carries an explicit misalignment marker.
func keywordSignal(entity entities.CanonicalEntity) (types.Mismatch, bool) {
	name := strings.ToLower(entity.Entity.Name)
	if !strings.Contains(name, "missing") && !strings.Contains(name, "mismatch") && !strings.Contains(name, "outdated") {
		return types.Mismatch{}, false
	}
	return types.Mismatch{
		ID:            fmt.Sprintf("keyword_signal:%s", entity.Entity.ID),
		Type:          "keyword_signal",
		Summary:       fmt.Sprintf("Potential delivery mismatch around %s", entity.Entity.Name),
		EntityIDs:     []string{entity.Entity.ID},
		Severity:      "medium",
		Confidence:    0.70,
		Impact:        "medium",
		Evidence:      entityEvidence(entity),
		AffectedRoles: []string{"engineering", "product"},
		Recommended:   "Confirm cross-layer understanding across all knowledge participants against the canonical context graph.",
	}, true
}

// requirementGap flags requirements that are not linked to any API or service that delivers them.
func requirementGap(entity entities.CanonicalEntity, outgoing map[string][]types.Relationship) (types.Mismatch, bool) {
	if entity.Entity.Type != types.Requirement {
		return types.Mismatch{}, false
	}
	for _, rel := range outgoing[entity.Entity.ID] {
		if rel.Kind == types.RequirementAffectsAPI || rel.Kind == types.RequirementAffectsService {
			return types.Mismatch{}, false
		}
	}
	return types.Mismatch{
		ID:            fmt.Sprintf("requirement_gap:%s", entity.Entity.ID),
		Type:          "requirement_gap",
		Summary:       fmt.Sprintf("Requirement %s is not linked to any API or service that delivers it", entity.Entity.Name),
		EntityIDs:     []string{entity.Entity.ID},
		Severity:      "high",
		Confidence:    0.65,
		Impact:        "high",
		Evidence:      entityEvidence(entity),
		AffectedRoles: []string{"product", "engineering"},
		Recommended:   "Trace this requirement to an owning API field or service, or confirm it is intentionally out of scope.",
	}, true
}

// crossLayerContractDrift flags API fields with explicit contract context that are not backed by any database column.
func crossLayerContractDrift(entity entities.CanonicalEntity, outgoing, incoming map[string][]types.Relationship) (types.Mismatch, bool) {
	if entity.Entity.Type != types.APIField {
		return types.Mismatch{}, false
	}
	if !hasContractExposure(entity.Entity.ID, outgoing, incoming) {
		return types.Mismatch{}, false
	}
	for _, rel := range outgoing[entity.Entity.ID] {
		if rel.Kind == types.APIBackedByDB {
			return types.Mismatch{}, false
		}
	}
	return types.Mismatch{
		ID:            fmt.Sprintf("cross_layer_contract_drift:%s", entity.Entity.ID),
		Type:          "cross_layer_contract_drift",
		Summary:       fmt.Sprintf("API field %s is exposed but not backed by any database column", entity.Entity.Name),
		EntityIDs:     []string{entity.Entity.ID},
		Severity:      "high",
		Confidence:    0.60,
		Impact:        "high",
		Evidence:      entityEvidence(entity),
		AffectedRoles: []string{"service", "consumer"},
		Recommended:   "Confirm the API field has persistent backing or update the contract so producers and consumers agree.",
	}, true
}

// hasContractExposure requires at least one typed relationship beyond raw co-occurrence
// so plain token mentions do not trigger cross-layer drift findings.
func hasContractExposure(entityID string, outgoing, incoming map[string][]types.Relationship) bool {
	for _, rel := range outgoing[entityID] {
		if rel.Kind != types.CoOccursInDocument && rel.Kind != types.APIBackedByDB {
			return true
		}
	}
	for _, rel := range incoming[entityID] {
		if rel.Kind != types.CoOccursInDocument && rel.Kind != types.APIBackedByDB {
			return true
		}
	}
	return false
}

// dependencyRisk surfaces service-to-dependency edges as delivery risk that needs an owner.
func dependencyRisk(rel types.Relationship) (types.Mismatch, bool) {
	if rel.Kind != types.ServiceDependsOn {
		return types.Mismatch{}, false
	}
	evidence := rel.Evidence
	if len(evidence) == 0 {
		evidence = []string{rel.ID}
	}
	return types.Mismatch{
		ID:            fmt.Sprintf("dependency_risk:%s", rel.ID),
		Type:          "dependency_risk",
		Summary:       fmt.Sprintf("Service %s depends on %s; confirm the dependency is healthy and owned", rel.FromID, rel.ToID),
		EntityIDs:     []string{rel.FromID, rel.ToID},
		Severity:      "medium",
		Confidence:    0.75,
		Impact:        "medium",
		Evidence:      evidence,
		AffectedRoles: []string{"service", "pmo"},
		Recommended:   "Verify the dependency owner, version, and availability before committing to delivery dates.",
	}, true
}

// outgoingByEntity groups relationships by their source entity ID for O(1) edge lookups.
func outgoingByEntity(rels []types.Relationship) map[string][]types.Relationship {
	index := map[string][]types.Relationship{}
	for _, rel := range rels {
		index[rel.FromID] = append(index[rel.FromID], rel)
	}
	return index
}

// incomingByEntity groups relationships by their target entity ID.
func incomingByEntity(rels []types.Relationship) map[string][]types.Relationship {
	index := map[string][]types.Relationship{}
	for _, rel := range rels {
		index[rel.ToID] = append(index[rel.ToID], rel)
	}
	return index
}

// entityEvidence builds a traceable source reference for an entity-derived finding.
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
