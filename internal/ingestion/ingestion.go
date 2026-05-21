package ingestion

import (
	"context" // carries cancellation signals into each connector

	"context-os/domain/contracts" // MCPSourceConnector interface and SourceRequest
	"context-os/domain/events"    // Event type collected from connectors
)

// Pipeline fans a single SourceRequest out to all registered connectors.
type Pipeline struct {
	connectors []contracts.MCPSourceConnector // ordered list of connectors to call during ingestion
}

// NewPipeline constructs a pipeline with the given connectors.
func NewPipeline(connectors ...contracts.MCPSourceConnector) Pipeline {
	return Pipeline{connectors: connectors} // store connectors for use in Ingest
}

// Ingest calls every connector in order and collects all emitted events into a single slice.
func (p Pipeline) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	var out []events.Event                       // accumulator for events from all connectors
	for _, connector := range p.connectors {     // iterate each registered connector in registration order
		ingested, err := connector.Ingest(ctx, req) // ask this connector to read the source
		if err != nil {
			return nil, err // stop on the first error so partial results are never silently returned
		}
		out = append(out, ingested...) // append this connector's events to the running total
	}
	return out, nil // return all collected events once every connector has run
}
