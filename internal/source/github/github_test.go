package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	githubsource "context-os/internal/source/github"
)

// TestNewConnectorExposesGitHubCapability verifies the GitHub connector exposes its expected identity and capability.
func TestNewConnectorExposesGitHubCapability(t *testing.T) {
	connector := githubsource.NewConnector()

	if connector.Name() != "github" {
		t.Fatalf("expected connector name github, got %q", connector.Name())
	}

	capabilities := connector.Capabilities()
	if len(capabilities) != 1 || capabilities[0] != contracts.CapabilityRepository {
		t.Fatalf("expected repository capability, got %#v", capabilities)
	}
}

// TestIngestDerivesRepositoryMetadataFromURI verifies repository URIs produce stable GitHub repository metadata.
func TestIngestDerivesRepositoryMetadataFromURI(t *testing.T) {
	connector := githubsource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "https://github.com/sx-tane/context-os",
		Content: "repository artifact",
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
	if event.SourceID != "github:repository:sx-tane/context-os" {
		t.Fatalf("expected repository source ID, got %q", event.SourceID)
	}

	wantMetadata := map[string]string{
		contracts.MetadataObjectType: "repository",
		contracts.MetadataObjectID:   "sx-tane/context-os",
		events.MetadataSourceID:      "github:repository:sx-tane/context-os",
		"github_owner":               "sx-tane",
		"github_repo":                "context-os",
	}
	for key, want := range wantMetadata {
		if event.Metadata[key] != want {
			t.Fatalf("metadata[%q] = %q, want %q", key, event.Metadata[key], want)
		}
	}
}

// TestIngestDerivesIssueAndPullRequestMetadata verifies issue and pull request URIs map to stable artifact metadata.
func TestIngestDerivesIssueAndPullRequestMetadata(t *testing.T) {
	tests := []struct {
		name            string
		uri             string
		wantObjectType  string
		wantObjectID    string
		wantSourceID    string
		wantIssueNumber string
	}{
		{
			name:            "issue URI",
			uri:             "repo://sx-tane/context-os/issues/7",
			wantObjectType:  "issue",
			wantObjectID:    "sx-tane/context-os#7",
			wantSourceID:    "github:issue:sx-tane/context-os#7",
			wantIssueNumber: "7",
		},
		{
			name:            "pull request URI",
			uri:             "https://github.com/sx-tane/context-os/pull/42",
			wantObjectType:  "pull_request",
			wantObjectID:    "sx-tane/context-os#42",
			wantSourceID:    "github:pull_request:sx-tane/context-os#42",
			wantIssueNumber: "42",
		},
	}

	connector := githubsource.NewConnector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := contracts.SourceRequest{URI: tt.uri, Content: "artifact"}

			ingested, err := connector.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("ingest returned error: %v", err)
			}
			event := ingested[0]

			if event.Metadata[contracts.MetadataObjectType] != tt.wantObjectType {
				t.Fatalf("object_type = %q, want %q", event.Metadata[contracts.MetadataObjectType], tt.wantObjectType)
			}
			if event.Metadata[contracts.MetadataObjectID] != tt.wantObjectID {
				t.Fatalf("object_id = %q, want %q", event.Metadata[contracts.MetadataObjectID], tt.wantObjectID)
			}
			if event.SourceID != tt.wantSourceID {
				t.Fatalf("source_id = %q, want %q", event.SourceID, tt.wantSourceID)
			}
			if event.Metadata["github_number"] != tt.wantIssueNumber {
				t.Fatalf("github_number = %q, want %q", event.Metadata["github_number"], tt.wantIssueNumber)
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

// TestIngestPreservesExplicitMetadataOverrides verifies caller-provided metadata takes precedence over derived values.
func TestIngestPreservesExplicitMetadataOverrides(t *testing.T) {
	connector := githubsource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "https://github.com/sx-tane/context-os/issues/7",
		Content: "artifact",
		Metadata: map[string]string{
			contracts.MetadataObjectType: "discussion",
			contracts.MetadataObjectID:   "custom-object-id",
			events.MetadataSourceID:      "custom-source-id",
		},
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]

	if event.Metadata[contracts.MetadataObjectType] != "discussion" {
		t.Fatalf("expected object_type override to be preserved, got %q", event.Metadata[contracts.MetadataObjectType])
	}
	if event.Metadata[contracts.MetadataObjectID] != "custom-object-id" {
		t.Fatalf("expected object_id override to be preserved, got %q", event.Metadata[contracts.MetadataObjectID])
	}
	if event.SourceID != "custom-source-id" {
		t.Fatalf("expected source_id override to be preserved, got %q", event.SourceID)
	}
}

// TestIngestFetchesArtifactContentFromGitHubAPI verifies empty-content requests hydrate from GitHub API with replay metadata.
func TestIngestFetchesArtifactContentFromGitHubAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/sx-tane/context-os/issues/7" {
			t.Fatalf("unexpected api path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if got := r.Header.Get("Accept"); !strings.Contains(got, "application/vnd.github+json") {
			t.Fatalf("unexpected accept header %q", got)
		}
		w.Header().Set("ETag", "\"etag-1\"")
		w.Header().Set("Last-Modified", "Mon, 26 May 2026 00:00:00 GMT")
		_, _ = w.Write([]byte(`{"id":7,"updated_at":"2026-05-26T00:00:00Z","title":"issue","body":"real content"}`))
	}))
	defer server.Close()

	connector := githubsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "repo://sx-tane/context-os/issues/7",
		Metadata: map[string]string{
			"github_api_base_url": server.URL,
			"github_token":        "token-123",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	if len(ingested) != 1 {
		t.Fatalf("expected one ingestion event, got %d", len(ingested))
	}

	event := ingested[0]
	if !strings.Contains(event.Content, `"id":7`) {
		t.Fatalf("expected raw payload content, got %q", event.Content)
	}
	if event.Metadata["github_api_status"] != "200" {
		t.Fatalf("expected github_api_status metadata, got %q", event.Metadata["github_api_status"])
	}
	if event.Metadata["github_etag"] != `"etag-1"` {
		t.Fatalf("expected github_etag metadata, got %q", event.Metadata["github_etag"])
	}
	if event.Metadata[contracts.MetadataSourceCursor] != "2026-05-26T00:00:00Z" {
		t.Fatalf("expected source cursor from updated_at, got %q", event.Metadata[contracts.MetadataSourceCursor])
	}
}

// TestIngestReturnsStructuredErrorWhenGitHubAPIFails verifies API failures include actionable object provenance.
func TestIngestReturnsStructuredErrorWhenGitHubAPIFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	connector := githubsource.NewConnector()
	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "repo://sx-tane/context-os/pulls/12",
		Metadata: map[string]string{
			"github_api_base_url": server.URL,
		},
	})
	if err == nil {
		t.Fatal("expected structured connector error")
	}

	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if connectorErr.ObjectType != "pull_request" {
		t.Fatalf("expected pull_request object type, got %q", connectorErr.ObjectType)
	}
	if connectorErr.ObjectID != "sx-tane/context-os#12" {
		t.Fatalf("expected pull request object ID, got %q", connectorErr.ObjectID)
	}
	if connectorErr.Kind != contracts.ErrorKindTemporary || !connectorErr.Retryable {
		t.Fatalf("expected retryable temporary error, got %#v", connectorErr)
	}
}
