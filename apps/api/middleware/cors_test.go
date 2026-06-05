package middleware_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/middleware"
)

// TestWithCORSSetsHeaders verifies WithCORS adds required headers for allowed local origins.
func TestWithCORSSetsHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.WithCORS(inner)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			req.Header.Set("Origin", "http://localhost:5173")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
				t.Errorf("Access-Control-Allow-Origin = %q, want localhost frontend", got)
			}
			if got := rec.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodDelete) {
				t.Errorf("Access-Control-Allow-Methods = %q, want DELETE included", got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-ContextOS-Request-ID") {
				t.Errorf("Access-Control-Allow-Headers = %q, want X-ContextOS-Request-ID", got)
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
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if called {
		t.Error("inner handler must not be called on OPTIONS preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want loopback frontend", got)
	}
}

// TestWithCORSRejectsUnknownOrigin verifies browser requests from non-local origins are rejected.
func TestWithCORSRejectsUnknownOrigin(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := middleware.WithCORS(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("inner handler called for rejected origin")
	}
}

// TestWithCORSAllowsConfiguredOrigin verifies CONTEXTOS_CORS_ORIGINS can add a custom local frontend origin.
func TestWithCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CONTEXTOS_CORS_ORIGINS", "http://localhost:3000")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	handler := middleware.WithCORS(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want configured origin", got)
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

// TestWithRequestLoggingIsQuietByDefault verifies request logging does not emit logs unless explicitly enabled.
func TestWithRequestLoggingIsQuietByDefault(t *testing.T) {
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middleware.WithRequestLogging("/quiet", inner)

	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/quiet", nil),
	)

	if got := logs.String(); got != "" {
		t.Fatalf("logs = %q, want empty", got)
	}
}

// TestWithRequestLoggingRecordsRequestLifecycle verifies enabled request logging includes start, completion, status, bytes, and request ID.
func TestWithRequestLoggingRecordsRequestLifecycle(t *testing.T) {
	t.Setenv("CONTEXTOS_API_REQUEST_LOGS", "1")
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, "created"); err != nil {
			t.Fatalf("WriteString() error = %v", err)
		}
	})
	handler := middleware.WithRequestLogging("/resource", inner)
	req := httptest.NewRequest(http.MethodPost, "/resource?debug=1", nil)
	req.Header.Set("X-ContextOS-Request-ID", "web-test-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	got := logs.String()
	for _, want := range []string{
		"http request start",
		"http request done",
		"id=web-test-1",
		"method=POST",
		"path=/resource",
		"route=/resource",
		"query=debug=1",
		"status=201",
		"bytes=7",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("logs missing %q: %s", want, got)
		}
	}
}

// TestWithRequestLoggingPreservesFlush verifies streaming handlers can still flush through the logging wrapper.
func TestWithRequestLoggingPreservesFlush(t *testing.T) {
	t.Setenv("CONTEXTOS_API_REQUEST_LOGS", "1")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}
		if _, err := io.WriteString(w, "event: log\n"); err != nil {
			t.Fatalf("WriteString() error = %v", err)
		}
		flusher.Flush()
	})
	handler := middleware.WithRequestLogging("/stream", inner)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/stream", nil))

	if got := rec.Body.String(); got != "event: log\n" {
		t.Fatalf("Body = %q, want event log line", got)
	}
	if !rec.Flushed {
		t.Fatal("response was not flushed")
	}
}
