package sharepoint

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns a SharePoint / OneDrive source connector that ingests files via Microsoft Graph.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("sharepoint", contracts.CapabilityFiles)
}
