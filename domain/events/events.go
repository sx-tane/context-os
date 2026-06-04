// Package events defines the serializable domain event envelope used by pipeline stages.
package events

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Type identifies the kind of event that occurred in the pipeline.
type Type string

const (
	// DocumentIngested identifies raw source artifacts captured by ingestion.
	DocumentIngested Type = "document.ingested"
	// DocumentNormalized identifies source artifacts converted to canonical documents.
	DocumentNormalized Type = "document.normalized"
	// EntityExtracted identifies entities pulled from normalized documents.
	EntityExtracted Type = "entity.extracted"
	// IdentityResolved identifies aliases merged into canonical identities.
	IdentityResolved Type = "identity.resolved"
	// RelationshipCreated identifies links created between domain entities.
	RelationshipCreated Type = "relationship.created"
	// MismatchDetected identifies delivery misalignments found by reasoning.
	MismatchDetected Type = "mismatch.detected"
	// CodexAnalysisComplete identifies completed local execution analysis.
	CodexAnalysisComplete Type = "codex.analysis.completed"
)

const (
	// SchemaVersion is the current version for the event envelope contract.
	SchemaVersion = "v1"
	// MetadataEventID names the metadata value used to provide a replay-stable event ID.
	MetadataEventID = "event_id"
	// MetadataSourceID names the metadata value used to provide source artifact identity.
	MetadataSourceID = "source_id"
	// MetadataTraceID names the metadata value used to correlate events in one pipeline run.
	MetadataTraceID = "trace_id"
)

// Event is the core unit of data flowing through the pipeline.
type Event struct {
	ID            string            `json:"id"             example:"evt_a1b2c3d4e5f6"`
	TraceID       string            `json:"trace_id"       example:"trace_a1b2c3d4e5f6"`
	Type          Type              `json:"type"           example:"document.ingested"`
	SchemaVersion string            `json:"schema_version" example:"v1"`
	Source        string            `json:"source"         example:"github"`
	SourceID      string            `json:"source_id"      example:"https://github.com/owner/repo/issues/1"`
	Subject       string            `json:"subject"        example:"Fix: update connector README"`
	Content       string            `json:"content"        example:"Issue body: Please update the connector README with setup steps."`
	Metadata      map[string]string `json:"metadata"`
	OccurredAt    time.Time         `json:"occurred_at"    example:"2026-05-29T10:00:00Z"`
}

// New creates an Event with replay-stable identity and provenance defaults.
func New(eventType Type, source, subject, content string, metadata map[string]string) Event {
	// Copy metadata so callers can mutate their input map without changing the event.
	metadata = cloneMetadata(metadata)

	// Prefer an explicit upstream source identifier, then fall back to subject/source.
	sourceID := strings.TrimSpace(metadata[MetadataSourceID])
	if sourceID == "" {
		sourceID = strings.TrimSpace(subject)
	}
	if sourceID == "" {
		sourceID = strings.TrimSpace(source)
	}

	// Reuse durable upstream event IDs when provided; otherwise derive deterministically.
	id := strings.TrimSpace(metadata[MetadataEventID])
	if id == "" {
		id = stableID(eventType, source, sourceID, subject)
	}

	// Keep trace IDs stable by default so a replay is easy to correlate downstream.
	traceID := strings.TrimSpace(metadata[MetadataTraceID])
	if traceID == "" {
		traceID = id
	}

	return Event{
		ID:            id,
		TraceID:       traceID,
		Type:          eventType,
		SchemaVersion: SchemaVersion,
		Source:        source,
		SourceID:      sourceID,
		Subject:       subject,
		Content:       content,
		Metadata:      metadata,
		OccurredAt:    time.Now().UTC(),
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	// Always return a non-nil map to keep downstream usage simple.
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func stableID(eventType Type, source, sourceID, subject string) string {
	parts := []string{
		string(eventType),
		strings.TrimSpace(source),
		strings.TrimSpace(sourceID),
		strings.TrimSpace(subject),
	}
	// Use an unambiguous delimiter before hashing so part boundaries stay stable.
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "event:" + hex.EncodeToString(sum[:])
}
