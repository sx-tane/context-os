package jira

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a Jira source connector that ingests issue tracker events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("jira", contracts.CapabilityIssues)
}
