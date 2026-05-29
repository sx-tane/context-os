package normalization_test

import (
	"testing"

	"context-os/domain/events"
	"context-os/internal/normalization"
)

// TestNormalizePreservesProvenanceAndCopiesMetadata verifies normalized documents keep event identity while isolating metadata from mutation.
func TestNormalizePreservesProvenanceAndCopiesMetadata(t *testing.T) {
	event := events.New(events.DocumentIngested, "github", "  repo://example  ", "  refund body  ", map[string]string{
		events.MetadataSourceID: "github:issue:1",
		"team":                  "payments",
	})

	got := normalization.Normalize(event)
	event.Metadata["team"] = "changed"

	if got.ID != event.ID {
		t.Fatalf("ID = %q, want %q", got.ID, event.ID)
	}
	if got.Source != "github" {
		t.Fatalf("Source = %q, want github", got.Source)
	}
	if got.SourceType != string(events.DocumentIngested) {
		t.Fatalf("SourceType = %q, want %q", got.SourceType, events.DocumentIngested)
	}
	if got.Title != "repo://example" {
		t.Fatalf("Title = %q, want repo://example", got.Title)
	}
	if got.Body != "refund body" {
		t.Fatalf("Body = %q, want refund body", got.Body)
	}
	if got.Metadata["team"] != "payments" {
		t.Fatalf("Metadata[team] = %q, want payments", got.Metadata["team"])
	}
	if got.NormalizedAt.IsZero() {
		t.Fatalf("NormalizedAt = %v, want timestamp", got.NormalizedAt)
	}
}
