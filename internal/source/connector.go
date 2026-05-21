package source

import (
	"context" // provides cancellation and deadline for each ingestion call
	"errors"  // used to construct a descriptive validation error
	"strings" // used to check whether the request content or URI is blank

	"github.com/sx-tane/context-os/domain/contracts" // Capability and SourceRequest types
	"github.com/sx-tane/context-os/domain/events"    // Event type emitted after successful ingestion
)

// MCPConnector is the shared base connector all source adapters are built on.
type MCPConnector struct {
	name         string               // human-readable identifier for this connector
	capabilities []contracts.Capability // list of data categories this connector supports
}

// NewMCPConnector creates a connector with the given name and set of capabilities.
func NewMCPConnector(name string, capabilities ...contracts.Capability) MCPConnector {
	return MCPConnector{name: name, capabilities: capabilities} // store name and capabilities for later use
}

// Name returns the connector's identifier so the pipeline can trace events back to their source.
func (c MCPConnector) Name() string { return c.name }

// Capabilities returns a copy of the capability list so callers cannot mutate the internal slice.
func (c MCPConnector) Capabilities() []contracts.Capability {
	out := make([]contracts.Capability, len(c.capabilities)) // allocate a new slice the same length
	copy(out, c.capabilities)                                 // copy values so the original slice is protected
	return out
}

// Ingest validates the request and emits a single DocumentIngested event.
func (c MCPConnector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // respect cancellation before doing any work
	}
	if strings.TrimSpace(req.Content) == "" && strings.TrimSpace(req.URI) == "" {
		return nil, errors.New("mcp source request requires content or uri") // reject empty requests immediately
	}
	metadata := map[string]string{"connector": c.name, "mcp": "true"} // seed metadata with connector identity
	for key, value := range req.Metadata {
		metadata[key] = value // merge any caller-supplied metadata on top
	}
	subject := req.URI // prefer the URI as the event subject because it is more stable than content
	if subject == "" {
		subject = c.name // fall back to the connector name when no URI is provided
	}
	return []events.Event{events.New(events.DocumentIngested, c.name, subject, req.Content, metadata)}, nil // wrap everything in a single ingestion event
}
