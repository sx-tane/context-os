package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"context-os/apps/api/middleware"
)

// TestWithCORSSetsHeaders verifies WithCORS adds all required CORS headers on GET and POST requests.
func TestWithCORSSetsHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.WithCORS(inner)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
				t.Error("Access-Control-Allow-Methods not set")
			}
			if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
				t.Error("Access-Control-Allow-Headers not set")
			}
		})
	}
}

// TestWithCORSOptionsPreflight verifies WithCORS returns 204 on OPTIONS and does not call the inner handler.
func TestWithCORSOptionsPreflight(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := middleware.WithCORS(inner)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if called {
		t.Error("inner handler must not be called on OPTIONS preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// TestWithCORSPassesRequestToInner verifies WithCORS forwards the request and inner handler status to the caller.
func TestWithCORSPassesRequestToInner(t *testing.T) {
	var receivedMethod string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusAccepted)
	})
	handler := middleware.WithCORS(inner)

	req := httptest.NewRequest(http.MethodPost, "/ingest", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if receivedMethod != http.MethodPost {
		t.Errorf("inner received method %q, want POST", receivedMethod)
	}
}
