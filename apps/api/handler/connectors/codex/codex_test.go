package codex_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"context-os/apps/api/handler/connectors/codex"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/codex/status", nil)

	codex.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReturnsJSON verifies that GET /codex/status returns 200 with a JSON body.
func TestStatusReturnsJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/codex/status", nil)

	codex.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	ct := recorder.Header().Get("Content-Type")
	if ct == "" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}

// TestSourcesMethodNotAllowed verifies that a non-GET request to Sources returns 405.
func TestSourcesMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/codex/sources?connector=github", nil)

	codex.Sources(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Sources() status = %d, want 405", recorder.Code)
	}
}

// TestSourcesRejectsUnknownConnector verifies that source discovery rejects unsupported connector names.
func TestSourcesRejectsUnknownConnector(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/codex/sources?connector=unknown", nil)

	codex.Sources(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Sources() status = %d, want 400", recorder.Code)
	}
}

// TestLoginMethodNotAllowed verifies that a non-POST request to Login returns 405.
func TestLoginMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/codex/login", nil)

	codex.Login(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Login() status = %d, want 405", recorder.Code)
	}
}

// TestPluginReauthMethodNotAllowed verifies that a non-POST request to PluginReauth returns 405.
func TestPluginReauthMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/codex/plugin-reauth", nil)

	codex.PluginReauth(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PluginReauth() status = %d, want 405", recorder.Code)
	}
}

// TestPluginReauthRejectsUnknownPlugin verifies that an unknown plugin name returns 400.
func TestPluginReauthRejectsUnknownPlugin(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/codex/plugin-reauth?plugin=unknown", nil)

	codex.PluginReauth(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("PluginReauth() status = %d, want 400", recorder.Code)
	}
}
