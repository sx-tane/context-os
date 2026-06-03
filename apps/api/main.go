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
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "context-os/apps/api/docs"
	handlerartifacts "context-os/apps/api/handler/artifacts"
	handlerchat "context-os/apps/api/handler/chat"
	handlercodex "context-os/apps/api/handler/codex"
	"context-os/apps/api/handler/filesystem"
	"context-os/apps/api/handler/github"
	googledrive "context-os/apps/api/handler/googledrive"
	handlergraph "context-os/apps/api/handler/graph"
	"context-os/apps/api/handler/health"
	"context-os/apps/api/handler/jira"
	notion "context-os/apps/api/handler/notion"
	presentation "context-os/apps/api/handler/presentation"
	"context-os/apps/api/handler/shared"
	sharepoint "context-os/apps/api/handler/sharepoint"
	"context-os/apps/api/handler/slack"
	handlerworkspace "context-os/apps/api/handler/workspace"
	"context-os/apps/api/middleware"
	"context-os/internal/aiworker"
	internalchat "context-os/internal/chat"
	"context-os/internal/execution"
	"context-os/internal/identity"
	"context-os/internal/normalization"
	"context-os/internal/store"
	internalsync "context-os/internal/sync"
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
	var presentationHandler *presentation.Handler
	var graphHandler *handlergraph.Handler
	var artifactsHandler *handlerartifacts.Handler
	var chatHandler *handlerchat.Handler
	if dbErr == nil {
		wsStore := store.NewWorkspaceStore(sqlDB)
		evStore := store.NewEventStore(sqlDB)
		syncStore := store.NewSyncStore(sqlDB)
		mismatchStore := store.NewMismatchStore(sqlDB)
		entityStore := store.NewEntityStore(sqlDB)
		auditStore := store.NewAuditStore(sqlDB)

		wsHandler = handlerworkspace.NewHandler(wsStore, evStore, entityStore, mismatchStore, syncStore).
			WithAuditRepository(auditStore)

		// Build optional persistence helpers that write to the local storage/ directories.
		embCache := aiworker.NewEmbeddingCache("storage/embeddings")
		aiClient := aiworker.New(aiworker.WithEmbeddingCache(embCache))
		parsedWriter := normalization.NewDocumentWriter("storage/parsed")
		tplExec := execution.TemplateExecutor{PromptsDir: "prompts"}

		presentationHandler = presentation.NewHandler(
			wsStore, evStore, mismatchStore, entityStore, syncStore,
			presentation.WithParsedWriter(parsedWriter),
			presentation.WithSemanticMatcher(identity.WorkerMatcher{Embedder: aiClient}),
			presentation.WithExecutor(tplExec),
			presentation.WithAuditRepository(auditStore),
		)
		shared.SetPersistentIngestService(shared.NewPersistentIngestService(
			wsStore, evStore, entityStore, mismatchStore, syncStore, auditStore,
			shared.WithPersistentParsedWriter(parsedWriter),
			shared.WithPersistentSemanticMatcher(identity.WorkerMatcher{Embedder: aiClient}),
		))
		graphHandler = handlergraph.NewHandler(wsStore, entityStore)
		artifactsHandler = handlerartifacts.NewHandler(wsStore, evStore)
		chatHandler = handlerchat.NewHandler(internalchat.NewServiceWithLiveAnswerer(wsStore, evStore, syncStore, internalchat.NewCodexAnswerer()))

		// Start background incremental sync worker.
		syncWorker := internalsync.NewWorker(wsStore, syncStore, evStore)
		go syncWorker.Run(context.Background(), 15*time.Minute)
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
		{pattern: "/github/ingest/stream", handler: http.HandlerFunc(github.IngestStream), cors: true},
		{pattern: "/github/status", handler: http.HandlerFunc(github.Status), cors: true},
		{pattern: "/jira/status", handler: http.HandlerFunc(jira.Status), cors: true},
		{pattern: "/jira/ingest", handler: http.HandlerFunc(jira.Ingest), cors: true},
		{pattern: "/jira/ingest/stream", handler: http.HandlerFunc(jira.IngestStream), cors: true},
		{pattern: "/filesystem/ingest", handler: http.HandlerFunc(filesystem.Ingest), cors: true},
		{pattern: "/filesystem/upload", handler: http.HandlerFunc(filesystem.Upload), cors: true},
		{pattern: "/codex/status", handler: http.HandlerFunc(handlercodex.Status), cors: true},
		{pattern: "/codex/sources", handler: http.HandlerFunc(handlercodex.Sources), cors: true},
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
			route{pattern: "/workspace", handler: http.HandlerFunc(wsHandler.Root), cors: true},
			route{pattern: "/workspace/upsert", handler: http.HandlerFunc(wsHandler.Upsert), cors: true},
			route{pattern: "/workspace/reset", handler: http.HandlerFunc(wsHandler.Reset), cors: true},
			route{pattern: "/workspace/status", handler: http.HandlerFunc(wsHandler.Status), cors: true},
		)
	}
	if presentationHandler != nil {
		routes = append(routes,
			route{pattern: "/presentation/findings", handler: http.HandlerFunc(presentationHandler.Findings), cors: true},
		)
	} else {
		routes = append(routes,
			route{pattern: "/presentation/findings", handler: http.HandlerFunc(presentation.Findings), cors: true},
		)
	}
	if graphHandler != nil {
		routes = append(routes,
			route{pattern: "/graph", handler: http.HandlerFunc(graphHandler.Query), cors: true},
		)
	}
	if artifactsHandler != nil {
		routes = append(routes,
			route{pattern: "/artifacts", handler: http.HandlerFunc(artifactsHandler.Query), cors: true},
		)
	}
	if chatHandler != nil {
		routes = append(routes,
			route{pattern: "/chat/query", handler: http.HandlerFunc(chatHandler.Query), cors: true},
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
