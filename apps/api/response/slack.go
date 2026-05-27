package response

import (
	"context-os/domain/events"
)

// SlackIngest is the JSON body returned by POST /slack/ingest.
type SlackIngest struct {
	Connector    string            `json:"connector"`
	Capabilities []string          `json:"capabilities"`
	Event        events.Event      `json:"event"`
	Preview      string            `json:"preview"`
	Metadata     map[string]string `json:"metadata"`
}
