package response

import (
	"context-os/domain/events"
)

// Ingest is the JSON body returned by ingest endpoints.
type Ingest struct {
	Connector    string            `json:"connector"`
	Capabilities []string          `json:"capabilities"`
	Event        events.Event      `json:"event"`
	Preview      string            `json:"preview"`
	Metadata     map[string]string `json:"metadata"`
}

// GithubIngest is the JSON body returned by POST /github/ingest.
type GithubIngest = Ingest
