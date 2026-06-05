package reasoning_test

import (
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/stages/reasoning"
)

type graphReader struct {
	entities      []entities.CanonicalEntity
	relationships []types.Relationship
}

func (r graphReader) AllEntities() []entities.CanonicalEntity {
	return r.entities
}

func (r graphReader) AllRelationships() []types.Relationship {
	return r.relationships
}

// TestDetectMismatchesEmitsEvidenceConfidenceAndImpact verifies keyword findings include traceable evidence and calibrated confidence.
func TestDetectMismatchesEmitsEvidenceConfidenceAndImpact(t *testing.T) {
	reader := graphReader{entities: []entities.CanonicalEntity{{
		Entity: types.Entity{
			ID:       "doc-1:missingrefundstate",
			Name:     "missingRefundState",
			SourceID: "doc-1",
			Metadata: map[string]string{contracts.MetadataSourceURI: "repo://refund-flow"},
		},
	}}}

	got := reasoning.DetectMismatches(reader)
	if len(got) != 1 {
		t.Fatalf("DetectMismatches() length = %d, want 1", len(got))
	}
	mismatch := got[0]
	if mismatch.Type != "keyword_signal" {
		t.Fatalf("Type = %q, want keyword_signal", mismatch.Type)
	}
	if mismatch.Confidence != 0.70 {
		t.Fatalf("Confidence = %v, want 0.70", mismatch.Confidence)
	}
	if mismatch.Impact != "medium" {
		t.Fatalf("Impact = %q, want medium", mismatch.Impact)
	}
	if len(mismatch.Evidence) != 1 || mismatch.Evidence[0] != "repo://refund-flow#missingRefundState" {
		t.Fatalf("Evidence = %v, want [repo://refund-flow#missingRefundState]", mismatch.Evidence)
	}
	if len(mismatch.AffectedRoles) == 0 {
		t.Fatalf("AffectedRoles = %v, want non-empty", mismatch.AffectedRoles)
	}
}

// TestDetectMismatchesIgnoresCleanEntities verifies entities without mismatch keywords or gaps produce no findings.
func TestDetectMismatchesIgnoresCleanEntities(t *testing.T) {
	reader := graphReader{
		entities: []entities.CanonicalEntity{
			{Entity: types.Entity{ID: "doc-1:refundstatus", Name: "refundStatus", Type: types.Enum}},
		},
	}

	got := reasoning.DetectMismatches(reader)
	if len(got) != 0 {
		t.Fatalf("DetectMismatches() length = %d, want 0", len(got))
	}
}

// TestDetectMismatchesGraphRules verifies cross-layer rules fire on real drift and stay silent when the graph is consistent.
func TestDetectMismatchesGraphRules(t *testing.T) {
	tests := []struct {
		name     string
		reader   graphReader
		wantType string
		wantLen  int
	}{
		{
			name: "standalone requirement without delivery evidence is not a gap",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "req-1", Name: "refundsMustBeReversible", Type: types.Requirement, SourceID: "doc-req"}},
				},
			},
			wantLen: 0,
		},
		{
			name: "requirement with same-source api but no owning edge is a gap",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "req-1", Name: "refundsMustBeReversible", Type: types.Requirement, SourceID: "doc-req"}},
					{Entity: types.Entity{ID: "api-1", Name: "refundReference", Type: types.APIField, SourceID: "doc-req"}},
				},
			},
			wantType: "requirement_gap",
			wantLen:  1,
		},
		{
			name: "requirement linked to an api is not a gap",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "req-1", Name: "refundsMustBeReversible", Type: types.Requirement, SourceID: "doc-req"}},
				},
				relationships: []types.Relationship{
					{ID: "req-1->api-1", FromID: "req-1", ToID: "api-1", Kind: types.RequirementAffectsAPI},
				},
			},
			wantLen: 0,
		},
		{
			name: "api field with contract exposure but no db backing drifts across layers",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "api-1", Name: "refundReference", Type: types.APIField, SourceID: "doc-api"}},
					{Entity: types.Entity{ID: "req-1", Name: "refundRequirement", Type: types.Requirement, SourceID: "doc-api"}},
				},
				relationships: []types.Relationship{
					{ID: "req-1->api-1", FromID: "req-1", ToID: "api-1", Kind: types.RequirementAffectsAPI},
				},
			},
			wantType: "cross_layer_contract_drift",
			wantLen:  1,
		},
		{
			name: "api field without graph context does not drift",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "api-1", Name: "refundReference", Type: types.APIField, SourceID: "doc-api"}},
				},
			},
			wantLen: 0,
		},
		{
			name: "api field backed by db does not drift",
			reader: graphReader{
				entities: []entities.CanonicalEntity{
					{Entity: types.Entity{ID: "api-1", Name: "refundReference", Type: types.APIField, SourceID: "doc-api"}},
				},
				relationships: []types.Relationship{
					{ID: "api-1->db-1", FromID: "api-1", ToID: "db-1", Kind: types.APIBackedByDB},
				},
			},
			wantLen: 0,
		},
		{
			name: "service dependency edge is a delivery risk",
			reader: graphReader{
				relationships: []types.Relationship{
					{ID: "svc-1->dep-1", FromID: "svc-1", ToID: "dep-1", Kind: types.ServiceDependsOn, Evidence: []string{"repo://svc#dep"}},
				},
			},
			wantType: "dependency_risk",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reasoning.DetectMismatches(tt.reader)
			if len(got) != tt.wantLen {
				t.Errorf("DetectMismatches() length = %d, want %d", len(got), tt.wantLen)
				return
			}
			if tt.wantLen == 0 {
				return
			}
			finding := got[0]
			if finding.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", finding.Type, tt.wantType)
			}
			if finding.Confidence <= 0 || finding.Confidence > 1 {
				t.Errorf("Confidence = %v, want within (0,1]", finding.Confidence)
			}
			if len(finding.Evidence) == 0 {
				t.Errorf("Evidence = %v, want non-empty", finding.Evidence)
			}
			if len(finding.AffectedRoles) == 0 {
				t.Errorf("AffectedRoles = %v, want non-empty", finding.AffectedRoles)
			}
			if finding.Recommended == "" {
				t.Errorf("Recommended = %q, want non-empty", finding.Recommended)
			}
		})
	}
}

// TestDetectMismatchesIsDeterministic verifies the same graph yields findings in stable ID order across runs.
func TestDetectMismatchesIsDeterministic(t *testing.T) {
	reader := graphReader{
		entities: []entities.CanonicalEntity{
			{Entity: types.Entity{ID: "api-1", Name: "refundReference", Type: types.APIField, SourceID: "doc-api"}},
			{Entity: types.Entity{ID: "req-1", Name: "refundsMustBeReversible", Type: types.Requirement, SourceID: "doc-req"}},
		},
		relationships: []types.Relationship{
			{ID: "svc-1->dep-1", FromID: "svc-1", ToID: "dep-1", Kind: types.ServiceDependsOn},
		},
	}

	first := reasoning.DetectMismatches(reader)
	second := reasoning.DetectMismatches(reader)
	if len(first) != len(second) {
		t.Fatalf("run lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("finding[%d].ID = %q on second run, want %q", i, second[i].ID, first[i].ID)
		}
	}
}
