// Package source provides shared MCP source connector behavior for concrete adapters.
package source

import (
	"context"
	"errors"
	"strings"

	"context-os/domain/contracts"
	"context-os/domain/events"
)

// MCPConnector is the shared base connector all source adapters are built on.
type MCPConnector struct {
	name         string
	capabilities []contracts.Capability
}

// NewMCPConnector creates a connector with the given name and set of capabilities.
func NewMCPConnector(name string, capabilities ...contracts.Capability) MCPConnector {
	return MCPConnector{name: name, capabilities: capabilities}
}

// Name returns the connector's identifier so the pipeline can trace events back to their source.
func (c MCPConnector) Name() string { return c.name }

// Capabilities returns a copy of the capability list so callers cannot mutate the internal slice.
func (c MCPConnector) Capabilities() []contracts.Capability {
	out := make([]contracts.Capability, len(c.capabilities))
	copy(out, c.capabilities)
	return out
}

// Ingest validates the request and emits a single DocumentIngested event.
func (c MCPConnector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	// Normalize once so validation, metadata, and subject all use the same trimmed value.
	uri := strings.TrimSpace(req.URI)
	cursor := strings.TrimSpace(req.Cursor)

	if strings.TrimSpace(req.Content) == "" && uri == "" {
		err := errors.New("source request requires content or uri")
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}

	// Merge caller metadata first, then overwrite with reserved keys so callers
	// cannot override connector, mcp, source_uri, or source_cursor.
	metadata := make(map[string]string, len(req.Metadata)+4)
	for key, value := range req.Metadata {
		if isSensitiveMetadataKey(key) {
			continue
		}
		metadata[key] = value
	}
	metadata[contracts.MetadataConnector] = c.name
	metadata[contracts.MetadataMCP] = "true"
	if uri != "" {
		metadata[contracts.MetadataSourceURI] = uri
	}
	if cursor != "" {
		metadata[contracts.MetadataSourceCursor] = cursor
	}

	subject := uri
	if subject == "" {
		subject = c.name
	}

	return []events.Event{events.New(events.DocumentIngested, c.name, subject, req.Content, metadata)}, nil
}

func isSensitiveMetadataKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "password") {
		return true
	}
	switch normalized {
	case "api_key",
		"apikey",
		"access_key",
		"private_key",
		"client_key",
		"googledrive_oauth_credentials_path",
		"googledrive_service_account_path",
		"credential_path",
		"service_account_path":
		return true
	default:
		return false
	}
}

func (c MCPConnector) connectorError(req contracts.SourceRequest, kind contracts.ErrorKind, retryable bool, err error) error {
	return &contracts.ConnectorError{
		Connector:  c.name,
		URI:        req.URI,
		ObjectType: req.Metadata[contracts.MetadataObjectType],
		ObjectID:   req.Metadata[contracts.MetadataObjectID],
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}
