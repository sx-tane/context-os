package events

import (
	"fmt"
	"time"
)

type Type string

const (
	DocumentIngested      Type = "document.ingested"
	DocumentNormalized    Type = "document.normalized"
	EntityExtracted       Type = "entity.extracted"
	IdentityResolved      Type = "identity.resolved"
	RelationshipCreated   Type = "relationship.created"
	MismatchDetected      Type = "mismatch.detected"
	CodexAnalysisComplete Type = "codex.analysis.completed"
)

type Event struct {
	ID         string            `json:"id"`
	Type       Type              `json:"type"`
	Source     string            `json:"source"`
	Subject    string            `json:"subject"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	OccurredAt time.Time         `json:"occurred_at"`
}

func New(eventType Type, source, subject, content string, metadata map[string]string) Event {
	if metadata == nil {
		metadata = map[string]string{}
	}
	occurredAt := time.Now().UTC()
	return Event{
		ID:         fmt.Sprintf("%s:%s:%d", eventType, subject, occurredAt.UnixNano()),
		Type:       eventType,
		Source:     source,
		Subject:    subject,
		Content:    content,
		Metadata:   metadata,
		OccurredAt: occurredAt,
	}
}
