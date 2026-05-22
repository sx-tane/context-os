package events

import (
	"encoding/json"
	"testing"
)

func TestEventVocabularyIncludesPipelineStages(t *testing.T) {
	// This table protects the event names other stages will publish and consume.
	want := map[Type]string{
		DocumentIngested:      "document.ingested",
		DocumentNormalized:    "document.normalized",
		EntityExtracted:       "entity.extracted",
		IdentityResolved:      "identity.resolved",
		RelationshipCreated:   "relationship.created",
		MismatchDetected:      "mismatch.detected",
		CodexAnalysisComplete: "codex.analysis.completed",
	}

	for eventType, value := range want {
		if string(eventType) != value {
			t.Fatalf("expected event type %q, got %q", value, eventType)
		}
	}
}

func TestNewCreatesReplayStableEnvelope(t *testing.T) {
	// Metadata can provide upstream provenance that should be copied into the envelope.
	metadata := map[string]string{
		MetadataSourceID: "github:issue:42",
		MetadataTraceID:  "trace-42",
		"team":           "payments",
	}

	// Creating the same event twice simulates replaying the same source artifact.
	event := New(DocumentIngested, "github", "repo#42", "refund status mismatch", metadata)
	replayed := New(DocumentIngested, "github", "repo#42", "refund status mismatch", metadata)
	metadata["team"] = "changed"

	if event.ID == "" {
		t.Fatal("expected stable event ID")
	}
	if event.ID != replayed.ID {
		t.Fatalf("expected replay-stable event ID, got %q and %q", event.ID, replayed.ID)
	}
	if event.TraceID != "trace-42" {
		t.Fatalf("expected trace ID from metadata, got %q", event.TraceID)
	}
	if event.SourceID != "github:issue:42" {
		t.Fatalf("expected source ID from metadata, got %q", event.SourceID)
	}
	if event.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, event.SchemaVersion)
	}
	if event.Source != "github" || event.Subject != "repo#42" || event.Content != "refund status mismatch" {
		t.Fatalf("event did not preserve source, subject, and content: %#v", event)
	}
	if event.Metadata["team"] != "payments" {
		t.Fatalf("expected metadata to be copied, got %q", event.Metadata["team"])
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("expected event timestamp")
	}
	if _, err := json.Marshal(event); err != nil {
		t.Fatalf("expected event to be JSON serializable: %v", err)
	}
}

func TestNewDefaultsReplayIdentifiers(t *testing.T) {
	// When callers do not pass metadata, New still creates stable IDs and a usable map.
	event := New(DocumentNormalized, "normalization", "document-1", "body", nil)
	replayed := New(DocumentNormalized, "normalization", "document-1", "body", nil)

	if event.ID != replayed.ID {
		t.Fatalf("expected default ID to be replay-stable, got %q and %q", event.ID, replayed.ID)
	}
	if event.TraceID != event.ID {
		t.Fatalf("expected default trace ID to match event ID, got %q and %q", event.TraceID, event.ID)
	}
	if event.SourceID != "document-1" {
		t.Fatalf("expected subject fallback source ID, got %q", event.SourceID)
	}
	if event.Metadata == nil {
		t.Fatal("expected non-nil metadata map")
	}
}

func TestNewAcceptsExplicitEventID(t *testing.T) {
	// Connectors may already have a durable event ID from the upstream system.
	event := New(EntityExtracted, "extraction", "document-1", "entity", map[string]string{
		MetadataEventID: "external-event-1",
	})

	if event.ID != "external-event-1" {
		t.Fatalf("expected explicit event ID, got %q", event.ID)
	}
	if event.TraceID != "external-event-1" {
		t.Fatalf("expected trace fallback to explicit event ID, got %q", event.TraceID)
	}
}
