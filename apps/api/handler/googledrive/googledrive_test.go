package googledrive_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/googledrive"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/googledrive/status", nil)

	googledrive.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReturnsDisconnectedWhenNoConfig verifies that Status reports disconnected when no Google Drive credentials are configured.
func TestStatusReturnsDisconnectedWhenNoConfig(t *testing.T) {
	t.Setenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH", "")
	t.Setenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH", "")
	t.Setenv("GOOGLE_DRIVE_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_DRIVE_FOLDER_ID", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/googledrive/status", nil)

	googledrive.Status(recorder, req)

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
	req := httptest.NewRequest(http.MethodGet, "/googledrive/ingest", nil)

	googledrive.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/googledrive/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	googledrive.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}
