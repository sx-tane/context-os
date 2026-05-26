package github_test

import (
	"context"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	githubsource "context-os/internal/source/github"
)

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
