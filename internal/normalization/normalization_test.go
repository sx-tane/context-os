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

// TestNormalizeDerivesStableContentHashAndSchemaVersion verifies canonical text yields a deterministic hash and carries the event schema version.
func TestNormalizeDerivesStableContentHashAndSchemaVersion(t *testing.T) {
	event := events.New(events.DocumentIngested, "github", "  refund title  ", "  refund body  ", nil)

	first := normalization.Normalize(event)
	second := normalization.Normalize(event)

	if first.ContentHash == "" {
		t.Fatalf("ContentHash = empty, want non-empty hash")
	}
	if first.ContentHash != second.ContentHash {
		t.Fatalf("ContentHash not deterministic: %q vs %q", first.ContentHash, second.ContentHash)
	}
	if first.SchemaVersion != event.SchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", first.SchemaVersion, event.SchemaVersion)
	}
	if first.RuleVersion != normalization.RuleVersion {
		t.Fatalf("RuleVersion = %q, want %q", first.RuleVersion, normalization.RuleVersion)
	}
}

// TestNormalizeReusesConnectorContentHash verifies a connector-provided content hash is preserved instead of recomputed.
func TestNormalizeReusesConnectorContentHash(t *testing.T) {
	event := events.New(events.DocumentIngested, "filesystem", "spec", "body", map[string]string{
		"filesystem_content_hash": "deadbeef",
	})

	got := normalization.Normalize(event)

	if got.ContentHash != "deadbeef" {
		t.Fatalf("ContentHash = %q, want deadbeef", got.ContentHash)
	}
}

// TestNormalizeRecordsSpansForCanonicalFields verifies non-empty title and body produce traceable rune-offset spans.
func TestNormalizeRecordsSpansForCanonicalFields(t *testing.T) {
	event := events.New(events.DocumentIngested, "github", "title", "longer body", nil)

	got := normalization.Normalize(event)

	if len(got.Spans) != 2 {
		t.Fatalf("Spans length = %d, want 2", len(got.Spans))
	}
	if got.Spans[0].Field != "title" || got.Spans[0].End != len("title") {
		t.Fatalf("title span = %+v, want field=title end=%d", got.Spans[0], len("title"))
	}
	if got.Spans[1].Field != "body" || got.Spans[1].End != len("longer body") {
		t.Fatalf("body span = %+v, want field=body end=%d", got.Spans[1], len("longer body"))
	}
}
