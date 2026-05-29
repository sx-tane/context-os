package jira_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/jira"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jira/status", nil)

	jira.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReportsEnvVarPresence verifies that Status reflects whether JIRA_BASE_URL is set.
func TestStatusReportsEnvVarPresence(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_TOKEN", "")
	t.Setenv("JIRA_EMAIL", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jira/status", nil)

	jira.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":false`) {
		t.Fatalf("body = %s, want connected:false when JIRA_BASE_URL is empty", body)
	}
}

// TestStatusReportsConnectedWhenBaseURLSet verifies that Status reports connected=true when JIRA_BASE_URL is configured.
func TestStatusReportsConnectedWhenBaseURLSet(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "https://example.atlassian.net")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jira/status", nil)

	jira.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":true`) {
		t.Fatalf("body = %s, want connected:true", body)
	}
}

// TestIngestMethodNotAllowed verifies that a non-POST request to Ingest returns 405.
func TestIngestMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jira/ingest", nil)

	jira.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jira/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	jira.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jira/ingest/stream", nil)

	jira.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
