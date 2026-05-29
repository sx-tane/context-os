// Package main is the entry point for the ContextOS HTTP API.
// It only wires routes to handlers; all logic lives in handler/, request/, response/, and middleware/.
//
// @title          ContextOS API
// @version        1.0
// @description    Local-first pipeline API for ingesting and reasoning over engineering context.
// @host           localhost:8080
// @BasePath       /
package main

import (
	"log"
	"net/http"
	"os"

	_ "context-os/apps/api/docs"
	"context-os/apps/api/handler"
	"context-os/apps/api/middleware"

	httpSwagger "github.com/swaggo/http-swagger"
)

// defaultAddr is the port the API binds to when API_ADDR is not set.
const defaultAddr = ":8080"

type route struct {
	pattern string
	handler http.Handler
	cors    bool
}

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	mux := http.NewServeMux()
	registerRoutes(mux, []route{
		{pattern: "/health", handler: http.HandlerFunc(handler.Health), cors: true},
		{pattern: "/github/ingest", handler: http.HandlerFunc(handler.GithubIngest), cors: true},
		{pattern: "/github/ingest/stream", handler: http.HandlerFunc(handler.GithubIngestStream), cors: true},
		{pattern: "/github/status", handler: http.HandlerFunc(handler.GithubStatus), cors: true},
		{pattern: "/jira/status", handler: http.HandlerFunc(handler.JiraStatus), cors: true},
		{pattern: "/jira/ingest", handler: http.HandlerFunc(handler.JiraIngest), cors: true},
		{pattern: "/jira/ingest/stream", handler: http.HandlerFunc(handler.JiraIngestStream), cors: true},
		{pattern: "/filesystem/ingest", handler: http.HandlerFunc(handler.FilesystemIngest), cors: true},
		{pattern: "/codex/status", handler: http.HandlerFunc(handler.CodexStatus), cors: true},
		{pattern: "/codex/login", handler: http.HandlerFunc(handler.CodexLogin), cors: true},
		{pattern: "/codex/plugin-reauth", handler: http.HandlerFunc(handler.CodexPluginReauth), cors: true},
		{pattern: "/slack/ingest", handler: http.HandlerFunc(handler.SlackIngest), cors: true},
		{pattern: "/slack/ingest/stream", handler: http.HandlerFunc(handler.SlackIngestStream), cors: true},
		{pattern: "/slack/status", handler: http.HandlerFunc(handler.SlackStatus), cors: true},
		{pattern: "/slack/connect", handler: http.HandlerFunc(handler.SlackConnect), cors: true},
		{pattern: "/slack/callback", handler: http.HandlerFunc(handler.SlackCallback)},
		{pattern: "/swagger/", handler: httpSwagger.WrapHandler},
	})

	log.Printf("context-os api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err)
	}
}

func registerRoutes(mux *http.ServeMux, routes []route) {
	for _, r := range routes {
		handler := r.handler
		if r.cors {
			handler = middleware.WithCORS(handler)
		}
		mux.Handle(r.pattern, handler)
	}
}
