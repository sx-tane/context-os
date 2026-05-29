package reasoning_test

import (
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/reasoning"
)

type entityReader struct {
	entities []entities.CanonicalEntity
}

func (r entityReader) AllEntities() []entities.CanonicalEntity {
	return r.entities
}

// TestDetectMismatchesEmitsEvidenceConfidenceAndImpact verifies keyword findings include traceable evidence and calibrated confidence.
func TestDetectMismatchesEmitsEvidenceConfidenceAndImpact(t *testing.T) {
	reader := entityReader{entities: []entities.CanonicalEntity{{
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
}

// TestDetectMismatchesIgnoresCleanEntities verifies entities without mismatch keywords produce no findings.
func TestDetectMismatchesIgnoresCleanEntities(t *testing.T) {
	reader := entityReader{entities: []entities.CanonicalEntity{{Entity: types.Entity{ID: "doc-1:refundstatus", Name: "refundStatus"}}}}

	got := reasoning.DetectMismatches(reader)
	if len(got) != 0 {
		t.Fatalf("DetectMismatches() length = %d, want 0", len(got))
	}
}
