package contracts

import (
	"context" // provides the cancellation and deadline context passed into connectors

	"context-os/domain/events" // event types produced by ingestion
)

// Capability describes a category of data a connector can ingest.
type Capability string

const (
	CapabilityRepository  Capability = "repository"  // connector can read code repositories
	CapabilityMessages    Capability = "messages"     // connector can read chat messages
	CapabilityIssues      Capability = "issues"       // connector can read issue trackers
	CapabilityAPISpec     Capability = "api_spec"     // connector can read API specifications
	CapabilitySpreadsheet Capability = "spreadsheet" // connector can read spreadsheets
	CapabilityFiles       Capability = "files"        // connector can read filesystem files
	CapabilityDocs        Capability = "docs"          // connector can read documentation pages
)

// SourceRequest carries the input a connector needs to locate and read a source artifact.
type SourceRequest struct {
	URI      string            `json:"uri"`      // address of the source artifact (URL, path, or identifier)
	Content  string            `json:"content"`  // raw content when the caller provides it directly
	Metadata map[string]string `json:"metadata"` // arbitrary key-value pairs that travel with the request
}

// MCPSourceConnector is the interface every source adapter must implement.
type MCPSourceConnector interface {
	Name() string                                                         // returns the connector's unique name
	Capabilities() []Capability                                           // returns what kinds of data this connector can provide
	Ingest(context.Context, SourceRequest) ([]events.Event, error)        // reads the source and emits ingestion events
}
