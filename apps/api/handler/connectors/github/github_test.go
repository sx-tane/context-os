package github_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/connectors/github"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/github/status", nil)

	github.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReturnsDisconnectedWhenNoToken verifies that Status reports connected=false when GITHUB_TOKEN is unset.
func TestStatusReturnsDisconnectedWhenNoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/github/status", nil)

	github.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":false`) {
		t.Fatalf("body = %s, want connected:false", body)
	}
}

// TestIngestMethodNotAllowed verifies that a non-POST request to Ingest returns 405.
func TestIngestMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/github/ingest", nil)

	github.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/github/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	github.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/github/ingest/stream", nil)

	github.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
