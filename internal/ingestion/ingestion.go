package ingestion

import (
	"context"

	"github.com/sx-tane/context-os/domain/contracts"
	"github.com/sx-tane/context-os/domain/events"
)

type Pipeline struct {
	connectors []contracts.MCPSourceConnector
}

func NewPipeline(connectors ...contracts.MCPSourceConnector) Pipeline {
	return Pipeline{connectors: connectors}
}

func (p Pipeline) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	var out []events.Event
	for _, connector := range p.connectors {
		ingested, err := connector.Ingest(ctx, req)
		if err != nil {
			return nil, err
		}
		out = append(out, ingested...)
	}
	return out, nil
}
