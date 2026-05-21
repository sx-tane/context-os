package filesystem

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a filesystem source connector that ingests local file events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("filesystem", contracts.CapabilityFiles)
}
