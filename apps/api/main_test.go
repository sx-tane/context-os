package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRegisterRoutesAppliesCORS verifies routes with cors:true receive CORS headers and routes with cors:false do not.
func TestRegisterRoutesAppliesCORS(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux, []route{
		{pattern: "/with-cors", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), cors: true},
		{pattern: "/no-cors", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), cors: false},
	})

	t.Run("cors route has Access-Control-Allow-Origin header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/with-cors", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}
	})

	t.Run("non-cors route has no Access-Control-Allow-Origin header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/no-cors", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})
}

// TestRegisterRoutesHandlesOPTIONS verifies cors-wrapped routes short-circuit OPTIONS preflight without calling the inner handler.
func TestRegisterRoutesHandlesOPTIONS(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	registerRoutes(mux, []route{
		{pattern: "/resource", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}), cors: true},
	})

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if called {
		t.Error("inner handler must not be called on OPTIONS preflight")
	}
}
