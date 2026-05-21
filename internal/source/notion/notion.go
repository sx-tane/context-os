package notion

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a Notion source connector that ingests pages and database entries.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("notion", contracts.CapabilityDocs)
}
