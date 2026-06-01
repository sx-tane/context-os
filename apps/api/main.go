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
	"database/sql"
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
	notion "context-os/apps/api/handler/notion"
	presentation "context-os/apps/api/handler/presentation"
	sharepoint "context-os/apps/api/handler/sharepoint"
	"context-os/apps/api/handler/slack"
	handlerworkspace "context-os/apps/api/handler/workspace"
	"context-os/apps/api/middleware"
	"context-os/internal/store"
	sqlmigrations "context-os/migrations"
	"context-os/storage/db"

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

	// Open DB and run migrations.  Failure is non-fatal so the API starts even
	// if Postgres is not yet available; workspace endpoints return 500 in that case.
	sqlDB, dbErr := openDB()

	mux := http.NewServeMux()

	var wsHandler *handlerworkspace.Handler
	if dbErr == nil {		wsHandler = handlerworkspace.NewHandler(
			store.NewWorkspaceStore(sqlDB),
			store.NewEventStore(sqlDB),
			store.NewSyncStore(sqlDB),
		)
	}

	routes := []route{
		{pattern: "/health", handler: http.HandlerFunc(health.Health), cors: true},
		{pattern: "/github/ingest", handler: http.HandlerFunc(github.Ingest), cors: true},
		{pattern: "/googledrive/status", handler: http.HandlerFunc(googledrive.Status), cors: true},
		{pattern: "/googledrive/ingest", handler: http.HandlerFunc(googledrive.Ingest), cors: true},
		{pattern: "/googledrive/ingest/stream", handler: http.HandlerFunc(googledrive.IngestStream), cors: true},
		{pattern: "/notion/status", handler: http.HandlerFunc(notion.Status), cors: true},
		{pattern: "/notion/ingest", handler: http.HandlerFunc(notion.Ingest), cors: true},
		{pattern: "/notion/ingest/stream", handler: http.HandlerFunc(notion.IngestStream), cors: true},
		{pattern: "/sharepoint/status", handler: http.HandlerFunc(sharepoint.Status), cors: true},
		{pattern: "/sharepoint/ingest", handler: http.HandlerFunc(sharepoint.Ingest), cors: true},
		{pattern: "/sharepoint/ingest/stream", handler: http.HandlerFunc(sharepoint.IngestStream), cors: true},
		{pattern: "/presentation/status", handler: http.HandlerFunc(presentation.Status), cors: true},
		{pattern: "/presentation/findings", handler: http.HandlerFunc(presentation.Findings), cors: true},
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
	}

	if wsHandler != nil {
		routes = append(routes,
			route{pattern: "/workspace", handler: http.HandlerFunc(wsHandler.List), cors: true},
			route{pattern: "/workspace/upsert", handler: http.HandlerFunc(wsHandler.Upsert), cors: true},
			route{pattern: "/workspace/status", handler: http.HandlerFunc(wsHandler.Status), cors: true},
		)
	}

	registerRoutes(mux, routes)

	log.Printf("context-os api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err)
	}
}

// openDB opens the Postgres connection pool and runs pending migrations.
func openDB() (*sql.DB, error) {
	conn, err := db.Open(sqlmigrations.Files)
	if err != nil {
		log.Printf("db: unavailable, workspace endpoints disabled: %v", err)
		return nil, err
	}
	log.Printf("db: connected and migrations applied")
	return conn, nil
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
