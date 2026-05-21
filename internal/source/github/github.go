package github

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a GitHub source connector that ingests code repository events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("github", contracts.CapabilityRepository)
}
