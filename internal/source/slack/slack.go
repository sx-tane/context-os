package slack

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a Slack source connector that ingests chat message events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("slack", contracts.CapabilityMessages)
}
