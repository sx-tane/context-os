// Package middleware provides reusable HTTP middleware for the ContextOS API.
package middleware

import (
	"net/http"
	"os"
	"strings"
)

const defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"

// WithCORS wraps a handler with local-only CORS headers so the SvelteKit
// frontend can call the API directly when not behind a proxy.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if !isAllowedOrigin(origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-ContextOS-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	for _, allowed := range allowedOrigins() {
		if origin == allowed {
			return true
		}
	}
	return false
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CONTEXTOS_CORS_ORIGINS"))
	if raw == "" {
		raw = defaultAllowedOrigins
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}
	return origins
}
