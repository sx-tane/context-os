package excel

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns an Excel source connector that ingests spreadsheet events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("excel", contracts.CapabilitySpreadsheet)
}
