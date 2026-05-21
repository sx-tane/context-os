package openapi

import (
	"context-os/domain/contracts"
	"context-os/internal/source"
)

// NewConnector returns an OpenAPI source connector that ingests API specification events.
func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("openapi", contracts.CapabilityAPISpec)
}
