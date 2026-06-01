package notion_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/domain/contracts"
	notionsource "context-os/internal/source/notion"
)

// TestNewConnectorExposesNotionCapability verifies the Notion connector identity and docs capability.
func TestNewConnectorExposesNotionCapability(t *testing.T) {
	c := notionsource.NewConnector()

	if c.Name() != "notion" {
		t.Fatalf("expected connector name notion, got %q", c.Name())
	}
	caps := c.Capabilities()
	if len(caps) != 1 || caps[0] != contracts.CapabilityDocs {
		t.Fatalf("expected docs capability, got %#v", caps)
	}
}

// TestIngestDerivesPageMetadataFromURI verifies notion://page/{id} URIs produce stable page metadata.
func TestIngestDerivesPageMetadataFromURI(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		wantType     string
		wantObjectID string
		wantSourceID string
	}{
		{
			name:         "hyphenated uuid",
			uri:          "notion://page/abc12345-1234-1234-1234-abcdefabcdef",
			wantType:     "page",
			wantObjectID: "abc12345-1234-1234-1234-abcdefabcdef",
		},
		{
			name:         "bare 32-hex id",
			uri:          "notion://page/abc1234512341234123412341234abcd",
			wantType:     "page",
			wantObjectID: "abc12345-1234-1234-1234-12341234abcd",
		},
		{
			name:         "database uri",
			uri:          "notion://database/deadbeef-cafe-babe-1234-000000000001",
			wantType:     "database",
			wantObjectID: "deadbeef-cafe-babe-1234-000000000001",
		},
	}

	c := notionsource.NewConnector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := contracts.SourceRequest{URI: tt.uri, Content: "content"}
			ingested, err := c.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}
			if len(ingested) == 0 {
				t.Fatalf("expected 1 event, got 0")
			}
			ev := ingested[0]
			if ev.Metadata[contracts.MetadataObjectType] != tt.wantType {
				t.Errorf("object_type = %q, want %q", ev.Metadata[contracts.MetadataObjectType], tt.wantType)
			}
			if tt.wantObjectID != "" && ev.Metadata[contracts.MetadataObjectID] != tt.wantObjectID {
				t.Errorf("object_id = %q, want %q", ev.Metadata[contracts.MetadataObjectID], tt.wantObjectID)
			}

			// Idempotency: same input must produce same event ID
			replayed, err := c.Ingest(context.Background(), req)
			if err != nil {
				t.Fatalf("replay Ingest() error = %v", err)
			}
			if replayed[0].ID != ev.ID {
				t.Errorf("replay event ID = %q, want %q", replayed[0].ID, ev.ID)
			}
		})
	}
}

// TestIngestFetchesPageContentFromNotionAPI verifies empty-content requests hydrate from the Notion API.
func TestIngestFetchesPageContentFromNotionAPI(t *testing.T) {
	pageID := "abc12345-1234-1234-1234-abcdefabcdef"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/pages/" + pageID:
			json.NewEncoder(w).Encode(map[string]any{
				"id":               pageID,
				"last_edited_time": "2026-05-01T00:00:00.000Z",
				"url":              "https://www.notion.so/Test-Page-" + strings.ReplaceAll(pageID, "-", ""),
				"properties": map[string]any{
					"title": map[string]any{
						"type": "title",
						"title": []any{
							map[string]any{"plain_text": "Test Page"},
						},
					},
				},
			})
		case "/blocks/" + pageID + "/children":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{
						"type": "paragraph",
						"paragraph": map[string]any{
							"rich_text": []any{
								map[string]any{"plain_text": "Hello Notion"},
							},
						},
					},
				},
				"has_more": false,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := notionsource.NewConnectorWithOptions(server.URL, &http.Client{})
	req := contracts.SourceRequest{
		URI:      "notion://page/" + pageID,
		Metadata: map[string]string{notionsource.MetadataToken: "testtoken"},
	}

	ingested, err := c.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) == 0 {
		t.Fatalf("expected 1 event, got 0")
	}
	ev := ingested[0]
	if !strings.Contains(ev.Content, "Hello Notion") {
		t.Errorf("Content = %q, want it to contain Hello Notion", ev.Content)
	}
	if ev.Metadata["notion_last_edited_time"] != "2026-05-01T00:00:00.000Z" {
		t.Errorf("notion_last_edited_time = %q, want 2026-05-01T00:00:00.000Z", ev.Metadata["notion_last_edited_time"])
	}
	if ev.Metadata["notion_title"] != "Test Page" {
		t.Errorf("notion_title = %q, want Test Page", ev.Metadata["notion_title"])
	}
}

// TestIngestRejectsUnsupportedURI verifies that an unrecognised URI returns an invalid_request error.
func TestIngestRejectsUnsupportedURI(t *testing.T) {
	c := notionsource.NewConnector()
	req := contracts.SourceRequest{URI: "https://example.com/notnotion"}

	_, err := c.Ingest(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for unsupported URI, got nil")
	}
	var ce *contracts.ConnectorError
	if !isConnectorError(err, &ce) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindInvalidRequest {
		t.Errorf("Kind = %q, want invalid_request", ce.Kind)
	}
}

// TestIngestRejectsMissingTokenWhenNoContent verifies that fetching without a token returns a permanent error.
func TestIngestRejectsMissingTokenWhenNoContent(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "")

	c := notionsource.NewConnector()
	req := contracts.SourceRequest{URI: "notion://page/abc12345-1234-1234-1234-abcdefabcdef"}

	_, err := c.Ingest(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error when no token, got nil")
	}
	var ce *contracts.ConnectorError
	if !isConnectorError(err, &ce) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindPermanent {
		t.Errorf("Kind = %q, want permanent", ce.Kind)
	}
}

// TestIngestRespectsContextCancellation verifies that a cancelled context produces a canceled error.
func TestIngestRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := notionsource.NewConnector()
	req := contracts.SourceRequest{URI: "notion://page/abc12345-1234-1234-1234-abcdefabcdef"}

	_, err := c.Ingest(ctx, req)
	if err == nil {
		t.Fatalf("expected error for cancelled context, got nil")
	}
	var ce *contracts.ConnectorError
	if !isConnectorError(err, &ce) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if ce.Kind != contracts.ErrorKindCanceled {
		t.Errorf("Kind = %q, want canceled", ce.Kind)
	}
}

// TestIngestUsesEnvTokenFallback verifies that NOTION_TOKEN env var is used when no metadata token is set.
func TestIngestUsesEnvTokenFallback(t *testing.T) {
	pageID := "00000000-0000-0000-0000-000000000001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer envtoken" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/pages/" + pageID:
			json.NewEncoder(w).Encode(map[string]any{
				"id": pageID, "last_edited_time": "2026-01-01T00:00:00Z", "url": "", "properties": map[string]any{},
			})
		case "/blocks/" + pageID + "/children":
			json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "has_more": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("NOTION_TOKEN", "envtoken")

	c := notionsource.NewConnectorWithOptions(server.URL, &http.Client{})
	req := contracts.SourceRequest{URI: "notion://page/" + pageID}

	ingested, err := c.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) == 0 {
		t.Fatalf("expected 1 event, got 0")
	}
}

func isConnectorError(err error, out **contracts.ConnectorError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*contracts.ConnectorError)
	if ok {
		*out = ce
	}
	return ok
}
