package sharepoint_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/domain/contracts"
	sharepointsource "context-os/internal/source/sharepoint"
)

// TestNewConnectorExposesSharePointCapability verifies the connector name and files capability.
func TestNewConnectorExposesSharePointCapability(t *testing.T) {
	c := sharepointsource.NewConnector()

	if c.Name() != "sharepoint" {
		t.Fatalf("expected connector name sharepoint, got %q", c.Name())
	}
	caps := c.Capabilities()
	if len(caps) != 1 || caps[0] != contracts.CapabilityFiles {
		t.Fatalf("expected files capability, got %#v", caps)
	}
}

// TestIngestParsesSharePointURIs verifies URI parsing for sites and drives.
func TestIngestParsesSharePointURIs(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		wantItemID  string
		wantSiteID  string
		wantDriveID string
	}{
		{
			name:       "site item uri",
			uri:        "sharepoint://sites/contoso.sharepoint.com,abc/items/item001",
			wantSiteID: "contoso.sharepoint.com,abc",
			wantItemID: "item001",
		},
		{
			name:        "drive item uri",
			uri:         "sharepoint://drives/drive-001/items/item-002",
			wantDriveID: "drive-001",
			wantItemID:  "item-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := sharepointsource.NewConnector()
			req := contracts.SourceRequest{URI: tt.uri, Content: "some content"}
			events, err := c.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}
			if len(events) == 0 {
				t.Fatalf("expected 1 event, got 0")
			}
			md := events[0].Metadata
			if tt.wantItemID != "" && md["sharepoint_item_id"] != tt.wantItemID {
				t.Errorf("sharepoint_item_id = %q, want %q", md["sharepoint_item_id"], tt.wantItemID)
			}
			if tt.wantSiteID != "" && md["sharepoint_site_id"] != tt.wantSiteID {
				t.Errorf("sharepoint_site_id = %q, want %q", md["sharepoint_site_id"], tt.wantSiteID)
			}
			if tt.wantDriveID != "" && md["sharepoint_drive_id"] != tt.wantDriveID {
				t.Errorf("sharepoint_drive_id = %q, want %q", md["sharepoint_drive_id"], tt.wantDriveID)
			}
		})
	}
}

// TestIngestRejectsUnsupportedURI verifies that an unrecognised URI returns an invalid_request error.
func TestIngestRejectsUnsupportedURI(t *testing.T) {
	c := sharepointsource.NewConnector()
	req := contracts.SourceRequest{URI: "https://example.com/not-sharepoint"}

	_, err := c.Ingest(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	ce, ok := err.(*contracts.ConnectorError)
	if !ok {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindInvalidRequest {
		t.Errorf("Kind = %q, want invalid_request", ce.Kind)
	}
}

// TestIngestRespectsContextCancellation verifies a cancelled context returns a canceled error.
func TestIngestRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := sharepointsource.NewConnector()
	req := contracts.SourceRequest{URI: "sharepoint://sites/mysiteID/items/item001"}

	_, err := c.Ingest(ctx, req)
	if err == nil {
		t.Fatalf("expected error for cancelled context, got nil")
	}
	ce, ok := err.(*contracts.ConnectorError)
	if !ok {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindCanceled {
		t.Errorf("Kind = %q, want canceled", ce.Kind)
	}
}

// TestIngestRejectsMissingCredentialsWhenNoContent verifies missing credentials return a permanent error.
func TestIngestRejectsMissingCredentialsWhenNoContent(t *testing.T) {
	t.Setenv("SHAREPOINT_ACCESS_TOKEN", "")
	t.Setenv("SHAREPOINT_TENANT_ID", "")
	t.Setenv("SHAREPOINT_CLIENT_ID", "")
	t.Setenv("SHAREPOINT_CLIENT_SECRET", "")

	c := sharepointsource.NewConnector()
	req := contracts.SourceRequest{URI: "sharepoint://sites/siteID/items/item001"}

	_, err := c.Ingest(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error when no credentials, got nil")
	}
	ce, ok := err.(*contracts.ConnectorError)
	if !ok {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindPermanent {
		t.Errorf("Kind = %q, want permanent", ce.Kind)
	}
}

// TestIngestFetchesItemFromGraphAPI verifies content hydration from Microsoft Graph.
func TestIngestFetchesItemFromGraphAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/content"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Hello SharePoint"))
		case strings.Contains(r.URL.Path, "/items/"):
			json.NewEncoder(w).Encode(map[string]any{
				"id":                      "item001",
				"name":                    "test.txt",
				"eTag":                    "etag-v1",
				"lastModifiedDateTime":    "2026-01-01T00:00:00Z",
				"file": map[string]any{"mimeType": "text/plain"},
				"size": 15,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := sharepointsource.NewConnectorWithOptions(server.URL, server.URL, &http.Client{})
	req := contracts.SourceRequest{
		URI:      "sharepoint://sites/siteID/items/item001",
		Metadata: map[string]string{sharepointsource.MetadataAccessToken: "testtoken"},
	}

	ingested, err := c.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) == 0 {
		t.Fatalf("expected 1 event, got 0")
	}
	ev := ingested[0]
	if !strings.Contains(ev.Content, "Hello SharePoint") {
		t.Errorf("Content = %q, want it to contain 'Hello SharePoint'", ev.Content)
	}
	if ev.Metadata["sharepoint_item_name"] != "test.txt" {
		t.Errorf("sharepoint_item_name = %q, want test.txt", ev.Metadata["sharepoint_item_name"])
	}
	if ev.Metadata["sharepoint_etag"] != "etag-v1" {
		t.Errorf("sharepoint_etag = %q, want etag-v1", ev.Metadata["sharepoint_etag"])
	}
}

// TestIngestIsIdempotent verifies that ingesting the same content twice produces the same event ID.
func TestIngestIsIdempotent(t *testing.T) {
	c := sharepointsource.NewConnector()
	req := contracts.SourceRequest{
		URI:     "sharepoint://drives/drive-001/items/item-002",
		Content: "stable file content",
	}

	a, err := c.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	b, err := c.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("replay Ingest() error = %v", err)
	}
	if a[0].ID != b[0].ID {
		t.Errorf("event IDs differ: %q vs %q", a[0].ID, b[0].ID)
	}
}
