package contracts

import (
	"context"

	"github.com/sx-tane/context-os/shared/events"
)

type Capability string

const (
	CapabilityRepository  Capability = "repository"
	CapabilityMessages    Capability = "messages"
	CapabilityIssues      Capability = "issues"
	CapabilityAPISpec     Capability = "api_spec"
	CapabilitySpreadsheet Capability = "spreadsheet"
	CapabilityFiles       Capability = "files"
)

type SourceRequest struct {
	URI      string            `json:"uri"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

type MCPSourceConnector interface {
	Name() string
	Capabilities() []Capability
	Ingest(context.Context, SourceRequest) ([]events.Event, error)
}
