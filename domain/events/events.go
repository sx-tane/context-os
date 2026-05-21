package events

import (
	"fmt"  // used to build deterministic event IDs
	"time" // used to capture when each event occurred
)

// Type identifies the kind of event that occurred in the pipeline.
type Type string

const (
	DocumentIngested      Type = "document.ingested"         // a raw source artifact was captured
	DocumentNormalized    Type = "document.normalized"       // a document was converted to canonical form
	EntityExtracted       Type = "entity.extracted"          // a named entity was pulled from a document
	IdentityResolved      Type = "identity.resolved"         // two or more aliases were merged into one entity
	RelationshipCreated   Type = "relationship.created"      // a link between two entities was established
	MismatchDetected      Type = "mismatch.detected"         // a delivery misalignment was found
	CodexAnalysisComplete Type = "codex.analysis.completed" // an AI analysis task finished
)

// Event is the core unit of data flowing through the pipeline.
type Event struct {
	ID         string            `json:"id"`          // deterministic identifier built from type, subject, and timestamp
	Type       Type              `json:"type"`        // what kind of event this is
	Source     string            `json:"source"`      // which connector or stage emitted this event
	Subject    string            `json:"subject"`     // the artifact this event is about (URI, key, or name)
	Content    string            `json:"content"`     // raw text payload of the artifact
	Metadata   map[string]string `json:"metadata"`    // arbitrary key-value context attached by the emitting stage
	OccurredAt time.Time         `json:"occurred_at"` // UTC timestamp of when this event was created
}

// New creates an Event with a deterministic ID derived from type, subject, and nanosecond timestamp.
func New(eventType Type, source, subject, content string, metadata map[string]string) Event {
	if metadata == nil {
		metadata = map[string]string{} // always initialise to avoid nil map panics downstream
	}
	occurredAt := time.Now().UTC() // capture the current UTC time before building the ID
	return Event{
		ID:         fmt.Sprintf("%s:%s:%d", eventType, subject, occurredAt.UnixNano()), // embed nanoseconds so IDs are unique even for the same subject
		Type:       eventType,  // store the event kind for routing downstream
		Source:     source,     // record which connector produced this event
		Subject:    subject,    // record what artifact this event describes
		Content:    content,    // carry the raw text forward for normalization
		Metadata:   metadata,   // attach any extra context from the source
		OccurredAt: occurredAt, // store the timestamp for replay and ordering
	}
}
