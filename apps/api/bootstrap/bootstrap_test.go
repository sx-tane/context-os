package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRegisterRoutesAppliesCORS verifies routes with CORS enabled receive CORS headers and routes without CORS do not.
func TestRegisterRoutesAppliesCORS(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, []Route{
		{Pattern: "/with-cors", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), CORS: true},
		{Pattern: "/no-cors", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), CORS: false},
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

// TestRegisterRoutesHandlesOPTIONS verifies CORS-wrapped routes short-circuit OPTIONS preflight without calling the inner handler.
func TestRegisterRoutesHandlesOPTIONS(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	RegisterRoutes(mux, []Route{
		{Pattern: "/resource", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}), CORS: true},
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

// TestRoutesWithoutDBKeepsPublicFallbackRoutes verifies DB-backed routes are omitted while fallback presentation remains available.
func TestRoutesWithoutDBKeepsPublicFallbackRoutes(t *testing.T) {
	routes := Routes(nil)
	patterns := make(map[string]bool, len(routes))
	for _, route := range routes {
		patterns[route.Pattern] = true
	}

	if !patterns["/health"] {
		t.Fatal("routes missing /health")
	}
	if !patterns["/presentation/findings"] {
		t.Fatal("routes missing fallback /presentation/findings")
	}
	if patterns["/workspace"] {
		t.Fatal("routes should not include DB-backed /workspace without a DB")
	}
}
