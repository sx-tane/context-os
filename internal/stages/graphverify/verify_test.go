package graphverify_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
	"context-os/internal/stages/graphverify"
)

// TestAcceptKeepsEvidencedCrossSourceRelationship verifies valid verifier edges are accepted with provenance metadata.
func TestAcceptKeepsEvidencedCrossSourceRelationship(t *testing.T) {
	snapshot := graphverify.Snapshot{
		TraceID: "trace-1",
		Events:  []repository.IngestEvent{{ID: "evt-1"}, {ID: "evt-2"}},
		Entities: []entities.CanonicalEntity{
			canonical("requirement-1", "BKGDEV-8551", types.Requirement, "evt-1"),
			canonical("service-1", "kkg_payment", types.Service, "evt-2"),
		},
	}

	got := graphverify.Accept(snapshot, "codex_cli", []types.Relationship{{
		FromID:     "requirement-1",
		ToID:       "service-1",
		Kind:       types.RequirementAffectsService,
		Confidence: 0.86,
		Evidence:   []string{"evt-1", "evt-2"},
	}})
	if len(got) != 1 {
		t.Fatalf("len(Accept()) = %d, want 1", len(got))
	}
	if got[0].Metadata[graphverify.MetadataProvider] != "codex_cli" {
		t.Fatalf("provider metadata = %q, want codex_cli", got[0].Metadata[graphverify.MetadataProvider])
	}
	if got[0].Metadata[graphverify.MetadataTraceID] != "trace-1" {
		t.Fatalf("trace metadata = %q, want trace-1", got[0].Metadata[graphverify.MetadataTraceID])
	}
}

// TestAcceptRejectsInventedOrWeakRelationships verifies verifier proposals must cite known evidence and enough confidence.
func TestAcceptRejectsInventedOrWeakRelationships(t *testing.T) {
	snapshot := graphverify.Snapshot{
		TraceID: "trace-1",
		Events:  []repository.IngestEvent{{ID: "evt-1"}},
		Entities: []entities.CanonicalEntity{
			canonical("requirement-1", "BKGDEV-8551", types.Requirement, "evt-1"),
			canonical("service-1", "kkg_payment", types.Service, "evt-2"),
		},
	}

	got := graphverify.Accept(snapshot, "codex_cli", []types.Relationship{
		{FromID: "missing", ToID: "service-1", Kind: types.RequirementAffectsService, Confidence: 0.9, Evidence: []string{"evt-1"}},
		{FromID: "requirement-1", ToID: "service-1", Kind: types.RequirementAffectsService, Confidence: 0.5, Evidence: []string{"evt-1"}},
		{FromID: "requirement-1", ToID: "service-1", Kind: types.RequirementAffectsService, Confidence: 0.9, Evidence: []string{"evt-missing"}},
	})
	if len(got) != 0 {
		t.Fatalf("len(Accept()) = %d, want 0", len(got))
	}
}

// TestParseOutputReadsStrictJSONLine verifies Codex verifier output is parsed from the required prefix line.
func TestParseOutputReadsStrictJSONLine(t *testing.T) {
	got, err := graphverify.ParseOutput(`notes
CONTEXTOS_GRAPH_VERIFY_JSON: {"relationships":[{"from":"a","to":"b","kind":"co_occurs_in_document","evidence_ids":["evt-1"],"confidence":0.8}]}`)
	if err != nil {
		t.Fatalf("ParseOutput() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(ParseOutput()) = %d, want 1", len(got))
	}
	if got[0].FromID != "a" || got[0].ToID != "b" {
		t.Fatalf("relationship endpoints = %q -> %q, want a -> b", got[0].FromID, got[0].ToID)
	}
}

// canonical returns one canonical entity for verifier tests.
func canonical(id, name string, entityType types.EntityType, sourceID string) entities.CanonicalEntity {
	return entities.CanonicalEntity{
		Entity: types.Entity{
			ID:       id,
			Name:     name,
			Type:     entityType,
			SourceID: sourceID,
		},
		Confidence: 0.9,
	}
}
