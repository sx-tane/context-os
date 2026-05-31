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
	handlercodex "context-os/apps/api/handler/codex"
	"context-os/apps/api/handler/filesystem"
	"context-os/apps/api/handler/github"
	googledrive "context-os/apps/api/handler/googledrive"
	"context-os/apps/api/handler/health"
	"context-os/apps/api/handler/jira"
	"context-os/apps/api/handler/slack"
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
		{pattern: "/health", handler: http.HandlerFunc(health.Health), cors: true},
		{pattern: "/github/ingest", handler: http.HandlerFunc(github.Ingest), cors: true},
		{pattern: "/googledrive/status", handler: http.HandlerFunc(googledrive.Status), cors: true},
		{pattern: "/googledrive/ingest", handler: http.HandlerFunc(googledrive.Ingest), cors: true},
		{pattern: "/googledrive/ingest/stream", handler: http.HandlerFunc(googledrive.IngestStream), cors: true},
		{pattern: "/github/ingest/stream", handler: http.HandlerFunc(github.IngestStream), cors: true},
		{pattern: "/github/status", handler: http.HandlerFunc(github.Status), cors: true},
		{pattern: "/jira/status", handler: http.HandlerFunc(jira.Status), cors: true},
		{pattern: "/jira/ingest", handler: http.HandlerFunc(jira.Ingest), cors: true},
		{pattern: "/jira/ingest/stream", handler: http.HandlerFunc(jira.IngestStream), cors: true},
		{pattern: "/filesystem/ingest", handler: http.HandlerFunc(filesystem.Ingest), cors: true},
		{pattern: "/filesystem/upload", handler: http.HandlerFunc(filesystem.Upload), cors: true},
		{pattern: "/codex/status", handler: http.HandlerFunc(handlercodex.Status), cors: true},
		{pattern: "/codex/login", handler: http.HandlerFunc(handlercodex.Login), cors: true},
		{pattern: "/codex/plugin-reauth", handler: http.HandlerFunc(handlercodex.PluginReauth), cors: true},
		{pattern: "/slack/ingest", handler: http.HandlerFunc(slack.Ingest), cors: true},
		{pattern: "/slack/ingest/stream", handler: http.HandlerFunc(slack.IngestStream), cors: true},
		{pattern: "/slack/status", handler: http.HandlerFunc(slack.Status), cors: true},
		{pattern: "/slack/connect", handler: http.HandlerFunc(slack.Connect), cors: true},
		{pattern: "/slack/callback", handler: http.HandlerFunc(slack.Callback)},
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
