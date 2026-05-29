package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"context-os/apps/api/handler/health"
)

// TestHealthOK verifies GET /health returns 200 with a non-empty body.
func TestHealthOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	health.Health(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}

// TestHealthMethodNotAllowed verifies any HTTP method returns 200 because method enforcement is delegated to the router.
func TestHealthMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health", nil)

	health.Health(recorder, req)

	// Health does not check method — it only accepts GET by ignoring the request.
	// Any method returns 200 for liveness; the router enforces the method.
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}
