package googledrive

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a Google Drive source connector that ingests Docs, Sheets, and Slides.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("googledrive", contracts.CapabilityFiles)
}
