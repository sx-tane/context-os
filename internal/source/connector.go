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
	if strings.TrimSpace(req.Content) == "" && strings.TrimSpace(req.URI) == "" {
		err := errors.New("source request requires content or uri")
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}

	metadata := map[string]string{
		contracts.MetadataConnector: c.name,
		contracts.MetadataMCP:       "true",
	}
	if req.URI != "" {
		metadata[contracts.MetadataSourceURI] = req.URI
	}
	if req.Cursor != "" {
		metadata[contracts.MetadataSourceCursor] = req.Cursor
	}
	for key, value := range req.Metadata {
		metadata[key] = value
	}

	subject := req.URI
	if subject == "" {
		subject = c.name
	}

	return []events.Event{events.New(events.DocumentIngested, c.name, subject, req.Content, metadata)}, nil
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
