package jira

import (
	"github.com/sx-tane/context-os/internal/source"
	"github.com/sx-tane/context-os/shared/contracts"
)

func NewConnector() contracts.MCPSourceConnector {
	return source.NewMCPConnector("jira", contracts.CapabilityIssues)
}
