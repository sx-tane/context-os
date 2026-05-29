package jira_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	jirasource "context-os/internal/source/jira"
)

// TestNewConnectorExposesJiraCapability verifies the Jira connector identity and capability.
func TestNewConnectorExposesJiraCapability(t *testing.T) {
	connector := jirasource.NewConnector()

	if connector.Name() != "jira" {
		t.Fatalf("expected connector name jira, got %q", connector.Name())
	}
	capabilities := connector.Capabilities()
	if len(capabilities) != 1 || capabilities[0] != contracts.CapabilityIssues {
		t.Fatalf("expected issues capability, got %#v", capabilities)
	}
}

// TestIngestDerivesIssueAndProjectMetadata verifies Jira URIs map to stable artifact metadata.
func TestIngestDerivesIssueAndProjectMetadata(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		wantType       string
		wantObjectID   string
		wantSourceID   string
		wantIssueKey   string
		wantProjectKey string
	}{
		{
			name:           "issue URI",
			uri:            "jira://CTX-42",
			wantType:       "issue",
			wantObjectID:   "CTX-42",
			wantSourceID:   "jira:issue:CTX-42",
			wantIssueKey:   "CTX-42",
			wantProjectKey: "CTX",
		},
		{
			name:           "project URI",
			uri:            "jira://project/CTX",
			wantType:       "project",
			wantObjectID:   "CTX",
			wantSourceID:   "jira:project:CTX",
			wantProjectKey: "CTX",
		},
	}

	connector := jirasource.NewConnector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := contracts.SourceRequest{URI: tt.uri, Content: "artifact"}
			ingested, err := connector.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("ingest returned error: %v", err)
			}
			event := ingested[0]
			if event.Metadata[contracts.MetadataObjectType] != tt.wantType {
				t.Fatalf("object_type = %q, want %q", event.Metadata[contracts.MetadataObjectType], tt.wantType)
			}
			if event.Metadata[contracts.MetadataObjectID] != tt.wantObjectID {
				t.Fatalf("object_id = %q, want %q", event.Metadata[contracts.MetadataObjectID], tt.wantObjectID)
			}
			if event.SourceID != tt.wantSourceID {
				t.Fatalf("source_id = %q, want %q", event.SourceID, tt.wantSourceID)
			}
			if event.Metadata["jira_issue_key"] != tt.wantIssueKey {
				t.Fatalf("jira_issue_key = %q, want %q", event.Metadata["jira_issue_key"], tt.wantIssueKey)
			}
			if event.Metadata["jira_project_key"] != tt.wantProjectKey {
				t.Fatalf("jira_project_key = %q, want %q", event.Metadata["jira_project_key"], tt.wantProjectKey)
			}

			replayed, err := connector.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("replay ingest returned error: %v", err)
			}
			if replayed[0].ID != event.ID {
				t.Fatalf("expected replay-stable event ID %q, got %q", event.ID, replayed[0].ID)
			}
		})
	}
}

// TestIngestFetchesIssueContentFromJiraAPI verifies empty-content requests hydrate from Jira REST with replay metadata.
func TestIngestFetchesIssueContentFromJiraAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/CTX-42" {
			t.Fatalf("unexpected api path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.Query().Get("expand"), "changelog") {
			t.Fatalf("expected changelog expand, got query %q", r.URL.RawQuery)
		}
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token-123"))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Fatalf("unexpected auth header %q", got)
		}
		_, _ = w.Write([]byte(`{"key":"CTX-42","fields":{"summary":"Refund state","updated":"2026-05-28T12:00:00.000+0000"},"changelog":{"histories":[]}}`))
	}))
	defer server.Close()

	connector := jirasource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "jira://CTX-42",
		Metadata: map[string]string{
			"jira_api_base_url": server.URL,
			"jira_email":        "user@example.com",
			"jira_token":        "token-123",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if !strings.Contains(event.Content, `"key":"CTX-42"`) {
		t.Fatalf("expected raw Jira payload content, got %q", event.Content)
	}
	if event.Metadata["jira_api_status"] != "200" {
		t.Fatalf("jira_api_status = %q, want 200", event.Metadata["jira_api_status"])
	}
	if event.Metadata["jira_updated"] != "2026-05-28T12:00:00.000+0000" {
		t.Fatalf("jira_updated = %q", event.Metadata["jira_updated"])
	}
	if event.Metadata[contracts.MetadataSourceCursor] != "2026-05-28T12:00:00.000+0000" {
		t.Fatalf("expected source cursor from Jira updated field, got %q", event.Metadata[contracts.MetadataSourceCursor])
	}
}

// TestIngestReturnsStructuredErrorWhenJiraAPIFails verifies failures include actionable issue provenance.
func TestIngestReturnsStructuredErrorWhenJiraAPIFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	connector := jirasource.NewConnector()
	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "jira://CTX-42",
		Metadata: map[string]string{
			"jira_api_base_url": server.URL,
		},
	})
	if err == nil {
		t.Fatal("expected structured connector error")
	}

	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if connectorErr.ObjectType != "issue" || connectorErr.ObjectID != "CTX-42" {
		t.Fatalf("unexpected connector error provenance: %#v", connectorErr)
	}
	if connectorErr.Kind != contracts.ErrorKindTemporary || !connectorErr.Retryable {
		t.Fatalf("expected retryable temporary error, got %#v", connectorErr)
	}
}

// TestIngestPreservesExplicitMetadataOverrides verifies caller-provided IDs are preserved.
func TestIngestPreservesExplicitMetadataOverrides(t *testing.T) {
	connector := jirasource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:     "jira://CTX-42",
		Content: "artifact",
		Metadata: map[string]string{
			contracts.MetadataObjectType: "initiative",
			contracts.MetadataObjectID:   "custom-ticket",
			events.MetadataSourceID:      "custom-source",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if event.Metadata[contracts.MetadataObjectType] != "initiative" {
		t.Fatalf("expected object type override, got %q", event.Metadata[contracts.MetadataObjectType])
	}
	if event.Metadata[contracts.MetadataObjectID] != "custom-ticket" {
		t.Fatalf("expected object id override, got %q", event.Metadata[contracts.MetadataObjectID])
	}
	if event.SourceID != "custom-source" {
		t.Fatalf("expected source id override, got %q", event.SourceID)
	}
}
