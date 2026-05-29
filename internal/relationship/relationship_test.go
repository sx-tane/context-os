package relationship_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/relationship"
)

// TestBuildLinksAdjacentEntitiesFromSameSource verifies co-occurrence relationships are created only within one source document.
func TestBuildLinksAdjacentEntitiesFromSameSource(t *testing.T) {
	input := []entities.CanonicalEntity{
		{Entity: types.Entity{ID: "doc-1:a", SourceID: "doc-1"}},
		{Entity: types.Entity{ID: "doc-1:b", SourceID: "doc-1"}},
		{Entity: types.Entity{ID: "doc-2:c", SourceID: "doc-2"}},
	}

	got := relationship.Build(input)
	if len(got) != 1 {
		t.Fatalf("Build() length = %d, want 1", len(got))
	}
	if got[0].ID != "doc-1:a->doc-1:b" {
		t.Fatalf("ID = %q, want doc-1:a->doc-1:b", got[0].ID)
	}
	if got[0].Kind != "co_occurs_in_document" {
		t.Fatalf("Kind = %q, want co_occurs_in_document", got[0].Kind)
	}
	if got[0].Metadata["source_id"] != "doc-1" {
		t.Fatalf("Metadata[source_id] = %q, want doc-1", got[0].Metadata["source_id"])
	}
}
