package response

import (
	"context-os/domain/events"
)

// Ingest is the JSON body returned by ingest endpoints.
type Ingest struct {
	Connector     string              `json:"connector"      example:"github"`
	Capabilities  []string            `json:"capabilities"`
	Event         events.Event        `json:"event"`
	Events        []events.Event      `json:"events,omitempty"`
	EventCount    int                 `json:"event_count"    example:"1"`
	Preview       string              `json:"preview"        example:"Issue #1: Fix connector README — requesting updated setup steps."`
	Previews      []string            `json:"previews,omitempty"`
	Metadata      map[string]string   `json:"metadata"`
	MetadataItems []map[string]string `json:"metadata_items,omitempty"`
}
