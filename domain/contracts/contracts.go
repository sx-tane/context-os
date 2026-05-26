// Package contracts defines source connector contracts in the domain layer.
package contracts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"context-os/domain/events"
)

// Capability describes a category of data a connector can ingest.
type Capability string

const (
	// CapabilityRepository identifies connectors that can read code repositories.
	CapabilityRepository Capability = "repository"
	// CapabilityMessages identifies connectors that can read chat messages.
	CapabilityMessages Capability = "messages"
	// CapabilityIssues identifies connectors that can read issue trackers.
	CapabilityIssues Capability = "issues"
	// CapabilityAPISpec identifies connectors that can read API specifications.
	CapabilityAPISpec Capability = "api_spec"
	// CapabilitySpreadsheet identifies connectors that can read spreadsheets.
	CapabilitySpreadsheet Capability = "spreadsheet"
	// CapabilityFiles identifies connectors that can read filesystem files.
	CapabilityFiles Capability = "files"
	// CapabilityDocs identifies connectors that can read documentation pages.
	CapabilityDocs Capability = "docs"
)

const (
	// MetadataConnector names the connector that produced an ingestion event.
	MetadataConnector = "connector"
	// MetadataMCP marks events produced through the MCP source connector contract.
	MetadataMCP = "mcp"
	// MetadataSourceURI preserves the replayable source resource URI.
	MetadataSourceURI = "source_uri"
	// MetadataSourceCursor preserves the source checkpoint or pagination cursor.
	MetadataSourceCursor = "source_cursor"
	// MetadataObjectType names the source artifact kind for actionable connector errors.
	MetadataObjectType = "object_type"
	// MetadataObjectID names the source artifact identifier for actionable connector errors.
	MetadataObjectID = "object_id"
)

// SourceRequest carries the input a connector needs to locate and read a source artifact.
type SourceRequest struct {
	URI      string            `json:"uri"`
	Content  string            `json:"content"`
	Cursor   string            `json:"cursor"`
	Metadata map[string]string `json:"metadata"`
}

// ErrorKind classifies connector failures for retry and operator decisions.
type ErrorKind string

const (
	// ErrorKindCanceled identifies a connector stopped by context cancellation or deadline.
	ErrorKindCanceled ErrorKind = "canceled"
	// ErrorKindInvalidRequest identifies source requests that cannot be ingested as provided.
	ErrorKindInvalidRequest ErrorKind = "invalid_request"
	// ErrorKindTemporary identifies transient connector failures that may succeed later.
	ErrorKindTemporary ErrorKind = "temporary"
	// ErrorKindPermanent identifies connector failures that require request or configuration changes.
	ErrorKindPermanent ErrorKind = "permanent"
)

// ConnectorError is a structured source connector error with retryability and provenance.
type ConnectorError struct {
	Connector  string
	URI        string
	ObjectType string
	ObjectID   string
	Kind       ErrorKind
	Retryable  bool
	Err        error
}

// Error returns an actionable connector error message.
func (e *ConnectorError) Error() string {
	if e == nil {
		return "connector error"
	}

	parts := []string{"connector " + quoteOrUnknown(e.Connector)}
	if e.URI != "" {
		parts = append(parts, "uri "+quoteOrUnknown(e.URI))
	}
	if e.ObjectType != "" {
		parts = append(parts, "object_type "+quoteOrUnknown(e.ObjectType))
	}
	if e.ObjectID != "" {
		parts = append(parts, "object_id "+quoteOrUnknown(e.ObjectID))
	}
	if e.Kind != "" {
		parts = append(parts, "kind "+quoteOrUnknown(string(e.Kind)))
	}
	if e.Retryable {
		parts = append(parts, "retryable true")
	} else {
		parts = append(parts, "retryable false")
	}
	if e.Err != nil {
		return strings.Join(parts, ", ") + ": " + e.Err.Error()
	}
	return strings.Join(parts, ", ")
}

// Unwrap returns the underlying connector failure.
func (e *ConnectorError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsRetryable reports whether err is a structured retryable connector failure.
func IsRetryable(err error) bool {
	var connectorErr *ConnectorError
	if errors.As(err, &connectorErr) {
		return connectorErr.Retryable
	}
	return false
}

// MCPSourceConnector is the interface every source adapter must implement.
type MCPSourceConnector interface {
	Name() string
	Capabilities() []Capability
	Ingest(context.Context, SourceRequest) ([]events.Event, error)
}

func quoteOrUnknown(value string) string {
	if value == "" {
		return "<unknown>"
	}
	return fmt.Sprintf("%q", value)
}
