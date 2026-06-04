package response

import (
	"context-os/domain/events"
)

// Ingest is the JSON body returned by ingest endpoints.
type Ingest struct {
	Connector           string              `json:"connector"      example:"github"`
	Capabilities        []string            `json:"capabilities"`
	Event               events.Event        `json:"event"`
	Events              []events.Event      `json:"events,omitempty"`
	EventCount          int                 `json:"event_count"    example:"1"`
	PersistedEventCount int                 `json:"persisted_event_count,omitempty"`
	EntityCount         int                 `json:"entity_count,omitempty"`
	RelationshipCount   int                 `json:"relationship_count,omitempty"`
	MismatchCount       int                 `json:"mismatch_count,omitempty"`
	WorkspaceID         string              `json:"workspace_id,omitempty"`
	PersistenceMode     string              `json:"persistence_mode,omitempty"`
	Preview             string              `json:"preview"        example:"Issue #1: Fix connector README — requesting updated setup steps."`
	Previews            []string            `json:"previews,omitempty"`
	Metadata            map[string]string   `json:"metadata"`
	MetadataItems       []map[string]string `json:"metadata_items,omitempty"`
}
