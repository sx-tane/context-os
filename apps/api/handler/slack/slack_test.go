package slack_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/slack"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/slack/status", nil)

	slack.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReturnsDisconnectedWhenNoToken verifies that Status reports connected=false when no token is configured.
func TestStatusReturnsDisconnectedWhenNoToken(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("SLACK_CLIENT_ID", "")
	t.Setenv("SLACK_CLIENT_SECRET", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/status", nil)

	slack.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":false`) {
		t.Fatalf("body = %s, want connected:false", body)
	}
	if !strings.Contains(body, `"oauth_available":false`) {
		t.Fatalf("body = %s, want oauth_available:false", body)
	}
}

// TestStatusReportsConnectedWhenEnvTokenSet verifies that Status reports connected=true and source=env when SLACK_BOT_TOKEN is set.
func TestStatusReportsConnectedWhenEnvTokenSet(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/status", nil)

	slack.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"connected":true`) {
		t.Fatalf("body = %s, want connected:true", body)
	}
	if !strings.Contains(body, `"source":"env"`) {
		t.Fatalf("body = %s, want source:env", body)
	}
}

// TestConnectMethodNotAllowed verifies that a non-GET request to Connect returns 405.
func TestConnectMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/slack/connect", nil)

	slack.Connect(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Connect() status = %d, want 405", recorder.Code)
	}
}

// TestConnectReturnsServiceUnavailableWhenNotConfigured verifies that Connect returns 503 when SLACK_CLIENT_ID is unset.
func TestConnectReturnsServiceUnavailableWhenNotConfigured(t *testing.T) {
	t.Setenv("SLACK_CLIENT_ID", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/connect", nil)

	slack.Connect(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("Connect() status = %d, want 503", recorder.Code)
	}
}

// TestCallbackMethodNotAllowed verifies that a non-GET request to Callback returns 405.
func TestCallbackMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/slack/callback", nil)

	slack.Callback(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Callback() status = %d, want 405", recorder.Code)
	}
}

// TestCallbackRejectsMissingState verifies that Callback returns 400 when the state parameter is absent.
func TestCallbackRejectsMissingState(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/callback?code=abc", nil)

	slack.Callback(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Callback() status = %d, want 400", recorder.Code)
	}
}

// TestCallbackRejectsInvalidState verifies that Callback returns 400 when the state token is not recognised.
func TestCallbackRejectsInvalidState(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/callback?code=abc&state=notvalid", nil)

	slack.Callback(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Callback() status = %d, want 400", recorder.Code)
	}
}

// TestIngestMethodNotAllowed verifies that a non-POST request to Ingest returns 405.
func TestIngestMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/ingest", nil)

	slack.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/slack/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	slack.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slack/ingest/stream", nil)

	slack.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
