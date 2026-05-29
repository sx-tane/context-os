package source_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
	filesystemsource "context-os/internal/source/filesystem"
	githubsource "context-os/internal/source/github"
	googledrivesource "context-os/internal/source/googledrive"
	jirasource "context-os/internal/source/jira"
	notionsource "context-os/internal/source/notion"
	sharepointsource "context-os/internal/source/sharepoint"
	slacksource "context-os/internal/source/slack"
)

// TestMCPConnectorExposesIdentityAndCapabilities verifies connector name and capability exposure are stable and defensive.
func TestMCPConnectorExposesIdentityAndCapabilities(t *testing.T) {
	connector := source.NewMCPConnector("github", contracts.CapabilityRepository)

	if connector.Name() != "github" {
		t.Fatalf("expected connector name github, got %q", connector.Name())
	}

	capabilities := connector.Capabilities()
	if len(capabilities) != 1 || capabilities[0] != contracts.CapabilityRepository {
		t.Fatalf("expected repository capability, got %#v", capabilities)
	}

	capabilities[0] = contracts.CapabilityMessages
	if connector.Capabilities()[0] != contracts.CapabilityRepository {
		t.Fatal("expected capabilities to be returned as a defensive copy")
	}
}

// TestMCPConnectorIngestEmitsDocumentIngestedEventWithProvenance verifies ingest emits a provenance-rich document.ingested event.
func TestMCPConnectorIngestEmitsDocumentIngestedEventWithProvenance(t *testing.T) {
	connector := source.NewMCPConnector("github", contracts.CapabilityRepository)
	req := contracts.SourceRequest{
		URI:     "repo://context-os/issues/3",
		Content: "build the MCP source connector interface",
		Cursor:  "issue-cursor-3",
		Metadata: map[string]string{
			events.MetadataSourceID: "github:issue:3",
			"team":                  "platform",
		},
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
		t.Fatalf("expected document.ingested event, got %q", event.Type)
	}
	if event.Source != "github" || event.Subject != req.URI || event.Content != req.Content {
		t.Fatalf("unexpected event provenance: %#v", event)
	}
	if event.SourceID != "github:issue:3" {
		t.Fatalf("expected source ID from metadata, got %q", event.SourceID)
	}

	wantMetadata := map[string]string{
		contracts.MetadataConnector:    "github",
		contracts.MetadataMCP:          "true",
		contracts.MetadataSourceURI:    req.URI,
		contracts.MetadataSourceCursor: req.Cursor,
		"team":                         "platform",
	}
	for key, want := range wantMetadata {
		if event.Metadata[key] != want {
			t.Fatalf("metadata[%q] = %q, want %q", key, event.Metadata[key], want)
		}
	}

	replayed, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("replay ingest returned error: %v", err)
	}
	if replayed[0].ID != event.ID {
		t.Fatalf("expected replay-stable event ID %q, got %q", event.ID, replayed[0].ID)
	}
}

// TestMCPConnectorIngestRejectsEmptyRequestWithStructuredError verifies validation failures return actionable ConnectorError values.
func TestMCPConnectorIngestRejectsEmptyRequestWithStructuredError(t *testing.T) {
	connector := source.NewMCPConnector("slack", contracts.CapabilityMessages)
	req := contracts.SourceRequest{
		URI: " ",
		Metadata: map[string]string{
			contracts.MetadataObjectType: "message",
			contracts.MetadataObjectID:   "C123:42",
		},
	}

	_, err := connector.Ingest(context.Background(), req)
	if err == nil {
		t.Fatal("expected empty request error")
	}

	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if connectorErr.Connector != "slack" || connectorErr.ObjectType != "message" || connectorErr.ObjectID != "C123:42" {
		t.Fatalf("unexpected connector error provenance: %#v", connectorErr)
	}
	if connectorErr.Kind != contracts.ErrorKindInvalidRequest || contracts.IsRetryable(err) {
		t.Fatalf("expected non-retryable invalid request error, got %#v", connectorErr)
	}
}

// TestMCPConnectorIngestRespectsCancellationWithStructuredError verifies context cancellation returns retryable canceled ConnectorError values.
func TestMCPConnectorIngestRespectsCancellationWithStructuredError(t *testing.T) {
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	connector := source.NewMCPConnector("jira", contracts.CapabilityIssues)
	_, err := connector.Ingest(ctx, contracts.SourceRequest{URI: "jira://PROJ-3"})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected wrapped deadline error, got %v", err)
	}

	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if connectorErr.Kind != contracts.ErrorKindCanceled || !contracts.IsRetryable(err) {
		t.Fatalf("expected retryable canceled error, got %#v", connectorErr)
	}
}

// TestRequiredSourceConnectorsImplementMCPContract verifies required connectors satisfy the MCP contract and emit ingestion events.
func TestRequiredSourceConnectorsImplementMCPContract(t *testing.T) {
	tests := []struct {
		name       string
		connector  contracts.MCPSourceConnector
		capability contracts.Capability
	}{
		{name: "github", connector: githubsource.NewConnector(), capability: contracts.CapabilityRepository},
		{name: "slack", connector: slacksource.NewConnector(), capability: contracts.CapabilityMessages},
		{name: "jira", connector: jirasource.NewConnector(), capability: contracts.CapabilityIssues},
		{name: "filesystem", connector: filesystemsource.NewConnector(), capability: contracts.CapabilityFiles},
		{name: "googledrive", connector: googledrivesource.NewConnector(), capability: contracts.CapabilityFiles},
		{name: "notion", connector: notionsource.NewConnector(), capability: contracts.CapabilityDocs},
		{name: "sharepoint", connector: sharepointsource.NewConnector(), capability: contracts.CapabilityFiles},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.connector.Name() != tt.name {
				t.Fatalf("expected connector name %q, got %q", tt.name, tt.connector.Name())
			}
			capabilities := tt.connector.Capabilities()
			if len(capabilities) != 1 || capabilities[0] != tt.capability {
				t.Fatalf("expected capability %q, got %#v", tt.capability, capabilities)
			}
			ingested, err := tt.connector.Ingest(context.Background(), contracts.SourceRequest{
				URI:     tt.name + "://artifact/1",
				Content: "artifact content",
			})
			if err != nil {
				t.Fatalf("ingest returned error: %v", err)
			}
			if len(ingested) != 1 || ingested[0].Type != events.DocumentIngested {
				t.Fatalf("expected one document.ingested event, got %#v", ingested)
			}
		})
	}
}

// TestMCPConnectorIngestReservedMetadataKeysCannotBeOverriddenByCaller verifies the connector overwrites reserved keys regardless of caller-supplied values.
func TestMCPConnectorIngestReservedMetadataKeysCannotBeOverriddenByCaller(t *testing.T) {
	connector := source.NewMCPConnector("github", contracts.CapabilityRepository)
	req := contracts.SourceRequest{
		URI:     "repo://context-os/issues/5",
		Content: "mcp contract enforcement",
		Metadata: map[string]string{
			contracts.MetadataConnector:    "attacker",
			contracts.MetadataMCP:          "false",
			contracts.MetadataSourceURI:    "evil://uri",
			contracts.MetadataSourceCursor: "evil-cursor",
		},
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	meta := ingested[0].Metadata
	if meta[contracts.MetadataConnector] != "github" {
		t.Errorf("connector overridden: got %q, want %q", meta[contracts.MetadataConnector], "github")
	}
	if meta[contracts.MetadataMCP] != "true" {
		t.Errorf("mcp overridden: got %q, want %q", meta[contracts.MetadataMCP], "true")
	}
	if meta[contracts.MetadataSourceURI] != req.URI {
		t.Errorf("source_uri overridden: got %q, want %q", meta[contracts.MetadataSourceURI], req.URI)
	}
}
