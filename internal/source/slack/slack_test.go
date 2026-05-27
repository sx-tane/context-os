package slack_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	slacksource "context-os/internal/source/slack"
)

// TestNewConnectorExposesSlackCapability verifies the Slack connector exposes its expected identity and capability.
func TestNewConnectorExposesSlackCapability(t *testing.T) {
	connector := slacksource.NewConnector()

	if connector.Name() != "slack" {
		t.Fatalf("expected connector name slack, got %q", connector.Name())
	}

	capabilities := connector.Capabilities()
	if len(capabilities) != 1 || capabilities[0] != contracts.CapabilityMessages {
		t.Fatalf("expected messages capability, got %#v", capabilities)
	}
}

// TestIngestDerivesChannelMetadataFromURI verifies channel URIs produce stable Slack channel metadata.
func TestIngestDerivesChannelMetadataFromURI(t *testing.T) {
	connector := slacksource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "slack://C1234567890",
		Content: "channel artifact",
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	if len(ingested) != 1 {
		t.Fatalf("expected one ingestion event, got %d", len(ingested))
	}

	event := ingested[0]
	if event.Type != events.DocumentIngested {
		t.Fatalf("expected document.ingested, got %q", event.Type)
	}
	if event.SourceID != "slack:channel:C1234567890" {
		t.Fatalf("expected channel source ID, got %q", event.SourceID)
	}

	wantMetadata := map[string]string{
		contracts.MetadataObjectType: "channel",
		contracts.MetadataObjectID:   "C1234567890",
		events.MetadataSourceID:      "slack:channel:C1234567890",
		"slack_channel_id":           "C1234567890",
	}
	for key, want := range wantMetadata {
		if event.Metadata[key] != want {
			t.Fatalf("metadata[%q] = %q, want %q", key, event.Metadata[key], want)
		}
	}
}

// TestIngestDerivesMessageMetadataAndReplayStability verifies message URIs map to stable artifact metadata and replay-stable IDs.
func TestIngestDerivesMessageMetadataAndReplayStability(t *testing.T) {
	connector := slacksource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "slack://C1234567890/1716518400.123456",
		Content: "message artifact",
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]

	if event.Metadata[contracts.MetadataObjectType] != "message" {
		t.Fatalf("object_type = %q, want message", event.Metadata[contracts.MetadataObjectType])
	}
	if event.Metadata[contracts.MetadataObjectID] != "C1234567890:1716518400.123456" {
		t.Fatalf("object_id = %q, want C1234567890:1716518400.123456", event.Metadata[contracts.MetadataObjectID])
	}
	if event.SourceID != "slack:message:C1234567890:1716518400.123456" {
		t.Fatalf("source_id = %q, want slack:message:C1234567890:1716518400.123456", event.SourceID)
	}
	if event.Metadata["slack_channel_id"] != "C1234567890" {
		t.Fatalf("slack_channel_id = %q, want C1234567890", event.Metadata["slack_channel_id"])
	}
	if event.Metadata["slack_ts"] != "1716518400.123456" {
		t.Fatalf("slack_ts = %q, want 1716518400.123456", event.Metadata["slack_ts"])
	}

	// Replay must produce the same event ID.
	replayed, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("replay ingest returned error: %v", err)
	}
	if replayed[0].ID != event.ID {
		t.Fatalf("expected replay-stable event ID %q, got %q", event.ID, replayed[0].ID)
	}
}

// TestIngestPreservesExplicitMetadataOverrides verifies caller-provided metadata takes precedence over derived values.
func TestIngestPreservesExplicitMetadataOverrides(t *testing.T) {
	connector := slacksource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "slack://C1234567890/1716518400.123456",
		Content: "artifact",
		Metadata: map[string]string{
			contracts.MetadataObjectType: "thread",
			contracts.MetadataObjectID:   "custom-object-id",
			events.MetadataSourceID:      "custom-source-id",
		},
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]

	if event.Metadata[contracts.MetadataObjectType] != "thread" {
		t.Fatalf("expected object_type override to be preserved, got %q", event.Metadata[contracts.MetadataObjectType])
	}
	if event.Metadata[contracts.MetadataObjectID] != "custom-object-id" {
		t.Fatalf("expected object_id override to be preserved, got %q", event.Metadata[contracts.MetadataObjectID])
	}
	if event.SourceID != "custom-source-id" {
		t.Fatalf("expected source_id override to be preserved, got %q", event.SourceID)
	}
}

// TestIngestFetchesMessageContentFromSlackAPI verifies empty-content requests hydrate from Slack API with replay cursor.
func TestIngestFetchesMessageContentFromSlackAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.history" {
			t.Fatalf("unexpected api path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test-token" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if got := r.URL.Query().Get("channel"); got != "C1234567890" {
			t.Fatalf("unexpected channel param %q", got)
		}
		if got := r.URL.Query().Get("latest"); got != "1716518400.123456" {
			t.Fatalf("unexpected latest param %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1716518400.123456","text":"hello world","user":"U123"}]}`))
	}))
	defer server.Close()

	connector := slacksource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "slack://C1234567890/1716518400.123456",
		Metadata: map[string]string{
			"slack_api_base_url": server.URL,
			"slack_token":        "xoxb-test-token",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	if len(ingested) != 1 {
		t.Fatalf("expected one ingestion event, got %d", len(ingested))
	}

	event := ingested[0]
	if !strings.Contains(event.Content, `"ts":"1716518400.123456"`) {
		t.Fatalf("expected raw payload content, got %q", event.Content)
	}
	if event.Metadata["slack_api_status"] != "200" {
		t.Fatalf("expected slack_api_status=200, got %q", event.Metadata["slack_api_status"])
	}
	if event.Metadata[contracts.MetadataSourceCursor] != "1716518400.123456" {
		t.Fatalf("expected source cursor from ts, got %q", event.Metadata[contracts.MetadataSourceCursor])
	}
}

// TestIngestReturnsStructuredErrorOnSlackAPIFailure verifies Slack ok:false responses produce actionable ConnectorError values.
func TestIngestReturnsStructuredErrorOnSlackAPIFailure(t *testing.T) {
	tests := []struct {
		name          string
		slackError    string
		wantKind      contracts.ErrorKind
		wantRetryable bool
	}{
		{"channel not found", "channel_not_found", contracts.ErrorKindPermanent, false},
		{"not authed", "not_authed", contracts.ErrorKindPermanent, false},
		{"rate limited", "ratelimited", contracts.ErrorKindTemporary, true},
		{"unknown error", "fatal_error", contracts.ErrorKindTemporary, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"ok":false,"error":"` + tt.slackError + `"}`))
			}))
			defer server.Close()

			connector := slacksource.NewConnector()
			_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
				URI: "slack://C1234567890/1716518400.123456",
				Metadata: map[string]string{
					"slack_api_base_url": server.URL,
				},
			})
			if err == nil {
				t.Fatal("expected structured connector error")
			}

			var connectorErr *contracts.ConnectorError
			if !errors.As(err, &connectorErr) {
				t.Fatalf("expected ConnectorError, got %T", err)
			}
			if connectorErr.ObjectType != "message" {
				t.Fatalf("expected message object type, got %q", connectorErr.ObjectType)
			}
			if connectorErr.ObjectID != "C1234567890:1716518400.123456" {
				t.Fatalf("expected message object ID, got %q", connectorErr.ObjectID)
			}
			if connectorErr.Kind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", connectorErr.Kind, tt.wantKind)
			}
			if connectorErr.Retryable != tt.wantRetryable {
				t.Fatalf("retryable = %v, want %v", connectorErr.Retryable, tt.wantRetryable)
			}
		})
	}
}
