package openapi

import (
	"github.com/sx-tane/context-os/internal/source"
	"github.com/sx-tane/context-os/domain/contracts"
)

func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("openapi", contracts.CapabilityAPISpec)
}
