package notion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/notion"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notion/status", nil)

	notion.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReportsDisconnectedWhenNoToken verifies Status returns connected=false when NOTION_TOKEN is unset.
func TestStatusReportsDisconnectedWhenNoToken(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notion/status", nil)

	notion.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":false`) {
		t.Errorf("body = %s, want connected:false when NOTION_TOKEN is empty", body)
	}
}

// TestStatusReportsConnectedWhenTokenSet verifies Status returns connected=true when NOTION_TOKEN is set.
func TestStatusReportsConnectedWhenTokenSet(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "secret_test_token")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notion/status", nil)

	notion.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":true`) {
		t.Errorf("body = %s, want connected:true", body)
	}
}

// TestIngestMethodNotAllowed verifies that a non-POST request to Ingest returns 405.
func TestIngestMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notion/ingest", nil)

	notion.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notion/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	notion.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestWithPreloadedContent verifies a page URI with content returns 200.
func TestIngestWithPreloadedContent(t *testing.T) {
	body := `{"uri":"notion://page/abc12345-1234-1234-1234-abcdefabcdef","content":"hello page"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notion/ingest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	notion.Ingest(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Ingest() status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notion/ingest/stream", nil)

	notion.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
