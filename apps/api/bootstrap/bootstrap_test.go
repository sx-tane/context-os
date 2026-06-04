package bootstrap

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"context-os/internal/stages/graphverify"

	_ "github.com/lib/pq"
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
	if patterns["/workspace/analysis-basket"] {
		t.Fatal("routes should not include DB-backed /workspace/analysis-basket without a DB")
	}
}

// TestRoutesWithDBIncludesChatSessionReset verifies DB-backed chat routes include workspace session reset.
func TestRoutesWithDBIncludesChatSessionReset(t *testing.T) {
	db, err := sql.Open("postgres", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	routes := Routes(db)
	for _, route := range routes {
		if route.Pattern == "/chat/session/reset" {
			return
		}
	}
	t.Fatal("routes missing /chat/session/reset")
}

// TestRelationshipAssistantFromEnvRequiresExplicitCodexFlag verifies AI relationship assistance is opt-in.
func TestRelationshipAssistantFromEnvRequiresExplicitCodexFlag(t *testing.T) {
	t.Setenv("CONTEXTOS_AI_RELATIONSHIPS", "")
	if got := relationshipAssistantFromEnv(); got != nil {
		t.Fatalf("relationshipAssistantFromEnv() = %#v, want nil", got)
	}

	t.Setenv("CONTEXTOS_AI_RELATIONSHIPS", "codex")
	if got := relationshipAssistantFromEnv(); got == nil {
		t.Fatal("relationshipAssistantFromEnv() = nil, want assistant")
	}
}

// TestGraphVerifierFromEnvUsesCodexProviderForDocumentedAndAliasModes verifies graph verifier env modes produce Codex provenance.
func TestGraphVerifierFromEnvUsesCodexProviderForDocumentedAndAliasModes(t *testing.T) {
	for _, mode := range []string{"codex", "data_analytics"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CONTEXTOS_GRAPH_VERIFIER", mode)
			got := graphVerifierFromEnv(nil, nil)
			if got == nil {
				t.Fatalf("graphVerifierFromEnv() = nil, want verifier")
			}
			if got.Assistant.Provider() != "codex_cli" {
				t.Fatalf("Provider() = %q, want codex_cli", got.Assistant.Provider())
			}
		})
	}

	t.Setenv("CONTEXTOS_GRAPH_VERIFIER", "")
	if got := graphVerifierFromEnv(nil, nil); got != nil {
		t.Fatalf("graphVerifierFromEnv() = %#v, want nil", got)
	}

	t.Setenv("CONTEXTOS_GRAPH_VERIFIER", "unsupported")
	if got := graphVerifierFromEnv(nil, nil); got != nil {
		t.Fatalf("graphVerifierFromEnv() = %#v, want nil", got)
	}

	if graphverify.NewCodexAssistant().Provider() != "codex_cli" {
		t.Fatal("NewCodexAssistant() provider must stay codex_cli")
	}
}
