package ingestion_test

import (
	"context"
	"errors"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/stages/ingestion"
)

type fakeConnector struct {
	name   string
	events []events.Event
	err    error
	called *bool
}

func (c fakeConnector) Name() string {
	return c.name
}

func (c fakeConnector) Capabilities() []contracts.Capability {
	return nil
}

func (c fakeConnector) Ingest(context.Context, contracts.SourceRequest) ([]events.Event, error) {
	if c.called != nil {
		*c.called = true
	}
	if c.err != nil {
		return nil, c.err
	}
	return c.events, nil
}

// TestPipelineIngestPreservesConnectorOrder verifies ingestion appends connector events in registration order.
func TestPipelineIngestPreservesConnectorOrder(t *testing.T) {
	pipe := ingestion.NewPipeline(
		fakeConnector{name: "github", events: []events.Event{{Source: "github"}}},
		fakeConnector{name: "slack", events: []events.Event{{Source: "slack"}}},
	)

	got, err := pipe.Ingest(context.Background(), contracts.SourceRequest{URI: "fixture://source"})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Ingest() length = %d, want 2", len(got))
	}
	if got[0].Source != "github" || got[1].Source != "slack" {
		t.Fatalf("sources = [%s %s], want [github slack]", got[0].Source, got[1].Source)
	}
}

// TestPipelineIngestStopsOnConnectorError verifies ingestion returns the first connector error without calling later connectors.
func TestPipelineIngestStopsOnConnectorError(t *testing.T) {
	called := false
	wantErr := errors.New("connector failed")
	pipe := ingestion.NewPipeline(
		fakeConnector{name: "github", err: wantErr},
		fakeConnector{name: "slack", called: &called},
	)

	_, err := pipe.Ingest(context.Background(), contracts.SourceRequest{URI: "fixture://source"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Ingest() error = %v, want %v", err, wantErr)
	}
	if called {
		t.Fatalf("second connector called = %v, want false", called)
	}
}
