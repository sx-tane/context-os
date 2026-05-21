package confluence

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a Confluence source connector that ingests pages from spaces.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("confluence", contracts.CapabilityDocs)
}
