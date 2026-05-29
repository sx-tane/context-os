package extraction_test

import (
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/domain/types"
	"context-os/internal/extraction"
)

// TestExtractDeduplicatesTokensAndPreservesEvidenceMetadata verifies extraction returns stable candidates with classification and source provenance.
func TestExtractDeduplicatesTokensAndPreservesEvidenceMetadata(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "doc-1",
			Body: "refundStatus refundStatus missingRefundState DBColumn PaymentFlag",
			Metadata: map[string]string{
				contracts.MetadataSourceURI: "repo://refund-flow",
				events.MetadataSourceID:     "github:issue:1",
			},
		},
		Classification: types.ConsumerConcern,
	}

	got := extraction.Extract(doc)
	if len(got) != 4 {
		t.Fatalf("Extract() length = %d, want 4", len(got))
	}
	if got[0].Name != "refundStatus" || got[0].Type != types.APIField {
		t.Fatalf("first entity = %#v, want refundStatus api_field", got[0])
	}
	if got[1].Name != "missingRefundState" || got[1].Type != types.APIField {
		t.Fatalf("second entity = %#v, want missingRefundState api_field", got[1])
	}
	if got[2].Type != types.DBColumn {
		t.Fatalf("third entity type = %q, want %q", got[2].Type, types.DBColumn)
	}
	if got[3].Type != types.Enum {
		t.Fatalf("fourth entity type = %q, want %q", got[3].Type, types.Enum)
	}
	if got[0].Metadata["classification"] != string(types.ConsumerConcern) {
		t.Fatalf("Metadata[classification] = %q, want %q", got[0].Metadata["classification"], types.ConsumerConcern)
	}
	if got[0].Metadata[contracts.MetadataSourceURI] != "repo://refund-flow" {
		t.Fatalf("Metadata[source_uri] = %q, want repo://refund-flow", got[0].Metadata[contracts.MetadataSourceURI])
	}
	if got[0].Metadata[events.MetadataSourceID] != "github:issue:1" {
		t.Fatalf("Metadata[source_id] = %q, want github:issue:1", got[0].Metadata[events.MetadataSourceID])
	}
}
