package sharepoint_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/sharepoint"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sharepoint/status", nil)

	sharepoint.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReportsDisconnectedWhenNoCredentials verifies Status returns connected=false when no credentials are set.
func TestStatusReportsDisconnectedWhenNoCredentials(t *testing.T) {
	t.Setenv("SHAREPOINT_ACCESS_TOKEN", "")
	t.Setenv("SHAREPOINT_TENANT_ID", "")
	t.Setenv("SHAREPOINT_CLIENT_ID", "")
	t.Setenv("SHAREPOINT_CLIENT_SECRET", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sharepoint/status", nil)

	sharepoint.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":false`) {
		t.Errorf("body = %s, want connected:false when no credentials are set", body)
	}
}

// TestStatusReportsConnectedWhenAccessTokenSet verifies Status returns connected=true when SHAREPOINT_ACCESS_TOKEN is set.
func TestStatusReportsConnectedWhenAccessTokenSet(t *testing.T) {
	t.Setenv("SHAREPOINT_ACCESS_TOKEN", "mytoken")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sharepoint/status", nil)

	sharepoint.Status(recorder, req)

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
	req := httptest.NewRequest(http.MethodGet, "/sharepoint/ingest", nil)

	sharepoint.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sharepoint/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	sharepoint.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestWithPreloadedContent verifies a sites URI with content returns 200.
func TestIngestWithPreloadedContent(t *testing.T) {
	body := `{"uri":"sharepoint://sites/siteID/items/item001","content":"hello sharepoint"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sharepoint/ingest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	sharepoint.Ingest(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Ingest() status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sharepoint/ingest/stream", nil)

	sharepoint.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
