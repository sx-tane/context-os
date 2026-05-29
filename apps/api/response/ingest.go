package response

import (
	"context-os/domain/events"
)

// Ingest is the JSON body returned by ingest endpoints.
type Ingest struct {
	Connector     string              `json:"connector"`
	Capabilities  []string            `json:"capabilities"`
	Event         events.Event        `json:"event"`
	Events        []events.Event      `json:"events,omitempty"`
	EventCount    int                 `json:"event_count"`
	Preview       string              `json:"preview"`
	Previews      []string            `json:"previews,omitempty"`
	Metadata      map[string]string   `json:"metadata"`
	MetadataItems []map[string]string `json:"metadata_items,omitempty"`
}
