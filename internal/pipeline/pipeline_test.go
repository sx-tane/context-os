package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/entities"
	"context-os/domain/events"
	"context-os/domain/types"
)

type failingRelationshipAssistant struct {
	calls int
}

// ProposeRelationships records the call and returns a deterministic failure.
func (f *failingRelationshipAssistant) ProposeRelationships(context.Context, types.NormalizedDocument, []entities.CanonicalEntity) ([]types.Relationship, error) {
	f.calls++
	return nil, errors.New("relationship assist unavailable")
}

// TestPersistenceContextSurvivesCanceledParent verifies persistence writes use an active detached timeout.
func TestPersistenceContextSurvivesCanceledParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	parentCancel()

	ctx, cancel := persistenceContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("persistenceContext() error = %v, want active context", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("persistenceContext() has no deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("persistenceContext() deadline = %s, want future deadline", deadline)
	}
}

// TestRunEventsDisablesRelationshipAssistantByDefault verifies nil stores keep the pipeline deterministic.
func TestRunEventsDisablesRelationshipAssistantByDefault(t *testing.T) {
	result := RunEvents(context.Background(), pipelineEventsFixture(), contracts.SourceRequest{URI: "repo://pipeline/default"}, nil)

	for _, rel := range result.Relationships {
		if rel.Metadata["assistive"] == "true" {
			t.Fatalf("relationship %q is assistive, want deterministic output only", rel.ID)
		}
	}
}

// TestRunEventsFallsBackWhenRelationshipAssistantFails verifies assistant errors do not change deterministic relationships.
func TestRunEventsFallsBackWhenRelationshipAssistantFails(t *testing.T) {
	baseline := RunEvents(context.Background(), pipelineEventsFixture(), contracts.SourceRequest{URI: "repo://pipeline/default"}, nil)
	assistant := &failingRelationshipAssistant{}

	got := RunEvents(context.Background(), pipelineEventsFixture(), contracts.SourceRequest{URI: "repo://pipeline/default"}, &Stores{
		RelationshipAssistant: assistant,
	})

	if assistant.calls != 1 {
		t.Fatalf("assistant calls = %d, want 1", assistant.calls)
	}
	if len(got.Relationships) != len(baseline.Relationships) {
		t.Fatalf("Relationships length = %d, want %d", len(got.Relationships), len(baseline.Relationships))
	}
}

// pipelineEventsFixture returns one event with Codex label entities for relationship tests.
func pipelineEventsFixture() []events.Event {
	body := `Checkout context.
CONTEXTOS_LABELS_JSON: {"entities":{"requirement":[{"name":"Checkout fee rule","evidence":"Checkout fee rule requires checkoutFeeAmount","confidence":0.9}],"api_field":[{"name":"checkoutFeeAmount","evidence":"Checkout fee rule requires checkoutFeeAmount","confidence":0.9}],"service":[],"dependency":[],"enum":[],"db_column":[]},"risks":[],"decisions":[],"status":[]}`
	return []events.Event{
		events.New(events.DocumentIngested, "test", "repo://pipeline/default", body, map[string]string{}),
	}
}
