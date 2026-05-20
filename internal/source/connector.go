package source

import (
	"context"
	"errors"
	"strings"

	"github.com/sx-tane/context-os/shared/contracts"
	"github.com/sx-tane/context-os/shared/events"
)

type MCPConnector struct {
	name         string
	capabilities []contracts.Capability
}

func NewMCPConnector(name string, capabilities ...contracts.Capability) MCPConnector {
	return MCPConnector{name: name, capabilities: capabilities}
}

func (c MCPConnector) Name() string { return c.name }

func (c MCPConnector) Capabilities() []contracts.Capability {
	out := make([]contracts.Capability, len(c.capabilities))
	copy(out, c.capabilities)
	return out
}

func (c MCPConnector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Content) == "" && strings.TrimSpace(req.URI) == "" {
		return nil, errors.New("mcp source request requires content or uri")
	}
	metadata := map[string]string{"connector": c.name, "mcp": "true"}
	for key, value := range req.Metadata {
		metadata[key] = value
	}
	subject := req.URI
	if subject == "" {
		subject = c.name
	}
	return []events.Event{events.New(events.DocumentIngested, c.name, subject, req.Content, metadata)}, nil
}
