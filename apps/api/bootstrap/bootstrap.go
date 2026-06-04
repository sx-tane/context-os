// Package bootstrap composes API handlers, route registration, and optional DB-backed services.
package bootstrap

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	handlerartifacts "context-os/apps/api/handler/artifacts"
	handlerchat "context-os/apps/api/handler/chat"
	handlercodex "context-os/apps/api/handler/connectors/codex"
	"context-os/apps/api/handler/connectors/filesystem"
	"context-os/apps/api/handler/connectors/github"
	googledrive "context-os/apps/api/handler/connectors/googledrive"
	"context-os/apps/api/handler/connectors/jira"
	notion "context-os/apps/api/handler/connectors/notion"
	sharepoint "context-os/apps/api/handler/connectors/sharepoint"
	"context-os/apps/api/handler/connectors/slack"
	handlergraph "context-os/apps/api/handler/graph"
	"context-os/apps/api/handler/health"
	presentation "context-os/apps/api/handler/presentation"
	"context-os/apps/api/handler/shared"
	handlerworkspace "context-os/apps/api/handler/workspace"
	"context-os/apps/api/middleware"
	"context-os/domain/repository"
	"context-os/internal/persistence/store"
	internalchat "context-os/internal/runtime/chat"
	internalsync "context-os/internal/runtime/sync"
	"context-os/internal/stages/execution"
	"context-os/internal/stages/graphverify"
	"context-os/internal/stages/identity"
	"context-os/internal/stages/normalization"
	"context-os/internal/stages/relationship"
	"context-os/internal/worker/aiworker"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Route describes one HTTP pattern, handler, and whether the handler receives CORS wrapping.
type Route struct {
	Pattern string
	Handler http.Handler
	CORS    bool
}

// NewMux builds the API mux and registers all public routes.
func NewMux(sqlDB *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterRoutes(mux, Routes(sqlDB))
	return mux
}

// Routes returns the full API route table. DB-backed routes are included only when sqlDB is non-nil.
func Routes(sqlDB *sql.DB) []Route {
	handlers := newHandlers(sqlDB)
	routes := []Route{
		{Pattern: "/health", Handler: http.HandlerFunc(health.Health), CORS: true},
		{Pattern: "/github/ingest", Handler: http.HandlerFunc(github.Ingest), CORS: true},
		{Pattern: "/googledrive/status", Handler: http.HandlerFunc(googledrive.Status), CORS: true},
		{Pattern: "/googledrive/ingest", Handler: http.HandlerFunc(googledrive.Ingest), CORS: true},
		{Pattern: "/googledrive/ingest/stream", Handler: http.HandlerFunc(googledrive.IngestStream), CORS: true},
		{Pattern: "/notion/status", Handler: http.HandlerFunc(notion.Status), CORS: true},
		{Pattern: "/notion/ingest", Handler: http.HandlerFunc(notion.Ingest), CORS: true},
		{Pattern: "/notion/ingest/stream", Handler: http.HandlerFunc(notion.IngestStream), CORS: true},
		{Pattern: "/sharepoint/status", Handler: http.HandlerFunc(sharepoint.Status), CORS: true},
		{Pattern: "/sharepoint/ingest", Handler: http.HandlerFunc(sharepoint.Ingest), CORS: true},
		{Pattern: "/sharepoint/ingest/stream", Handler: http.HandlerFunc(sharepoint.IngestStream), CORS: true},
		{Pattern: "/presentation/status", Handler: http.HandlerFunc(presentation.Status), CORS: true},
		{Pattern: "/github/ingest/stream", Handler: http.HandlerFunc(github.IngestStream), CORS: true},
		{Pattern: "/github/status", Handler: http.HandlerFunc(github.Status), CORS: true},
		{Pattern: "/jira/status", Handler: http.HandlerFunc(jira.Status), CORS: true},
		{Pattern: "/jira/ingest", Handler: http.HandlerFunc(jira.Ingest), CORS: true},
		{Pattern: "/jira/ingest/stream", Handler: http.HandlerFunc(jira.IngestStream), CORS: true},
		{Pattern: "/filesystem/ingest", Handler: http.HandlerFunc(filesystem.Ingest), CORS: true},
		{Pattern: "/filesystem/upload", Handler: http.HandlerFunc(filesystem.Upload), CORS: true},
		{Pattern: "/codex/status", Handler: http.HandlerFunc(handlercodex.Status), CORS: true},
		{Pattern: "/codex/sources", Handler: http.HandlerFunc(handlercodex.Sources), CORS: true},
		{Pattern: "/codex/login", Handler: http.HandlerFunc(handlercodex.Login), CORS: true},
		{Pattern: "/codex/plugin-reauth", Handler: http.HandlerFunc(handlercodex.PluginReauth), CORS: true},
		{Pattern: "/slack/ingest", Handler: http.HandlerFunc(slack.Ingest), CORS: true},
		{Pattern: "/slack/ingest/stream", Handler: http.HandlerFunc(slack.IngestStream), CORS: true},
		{Pattern: "/slack/status", Handler: http.HandlerFunc(slack.Status), CORS: true},
		{Pattern: "/slack/connect", Handler: http.HandlerFunc(slack.Connect), CORS: true},
		{Pattern: "/slack/callback", Handler: http.HandlerFunc(slack.Callback)},
		{Pattern: "/swagger/", Handler: httpSwagger.WrapHandler},
	}

	if handlers.workspace != nil {
		routes = append(routes,
			Route{Pattern: "/workspace", Handler: http.HandlerFunc(handlers.workspace.Root), CORS: true},
			Route{Pattern: "/workspace/upsert", Handler: http.HandlerFunc(handlers.workspace.Upsert), CORS: true},
			Route{Pattern: "/workspace/source", Handler: http.HandlerFunc(handlers.workspace.Source), CORS: true},
			Route{Pattern: "/workspace/reset", Handler: http.HandlerFunc(handlers.workspace.Reset), CORS: true},
			Route{Pattern: "/workspace/status", Handler: http.HandlerFunc(handlers.workspace.Status), CORS: true},
			Route{Pattern: "/workspace/analysis-basket", Handler: http.HandlerFunc(handlers.workspace.AnalysisBasket), CORS: true},
			Route{Pattern: "/workspace/finding-actions", Handler: http.HandlerFunc(handlers.workspace.FindingActions), CORS: true},
		)
	}
	if handlers.presentation != nil {
		routes = append(routes,
			Route{Pattern: "/presentation/findings", Handler: http.HandlerFunc(handlers.presentation.Findings), CORS: true},
		)
	} else {
		routes = append(routes,
			Route{Pattern: "/presentation/findings", Handler: http.HandlerFunc(presentation.Findings), CORS: true},
		)
	}
	if handlers.graph != nil {
		routes = append(routes,
			Route{Pattern: "/graph", Handler: http.HandlerFunc(handlers.graph.Query), CORS: true},
			Route{Pattern: "/graph/cleanup", Handler: http.HandlerFunc(handlers.graph.Cleanup), CORS: true},
		)
	}
	if handlers.artifacts != nil {
		routes = append(routes,
			Route{Pattern: "/artifacts", Handler: http.HandlerFunc(handlers.artifacts.Query), CORS: true},
			Route{Pattern: "/artifacts/live-evidence/cleanup", Handler: http.HandlerFunc(handlers.artifacts.CleanupLiveEvidence), CORS: true},
		)
	}
	if handlers.chat != nil {
		routes = append(routes,
			Route{Pattern: "/chat/query", Handler: http.HandlerFunc(handlers.chat.Query), CORS: true},
			Route{Pattern: "/chat/query/stream", Handler: http.HandlerFunc(handlers.chat.StreamQuery), CORS: true},
			Route{Pattern: "/chat/session/reset", Handler: http.HandlerFunc(handlers.chat.ResetSession), CORS: true},
		)
	}
	return routes
}

// RegisterRoutes mounts routes onto mux, applying CORS middleware where requested.
func RegisterRoutes(mux *http.ServeMux, routes []Route) {
	for _, r := range routes {
		handler := r.Handler
		if r.CORS {
			handler = middleware.WithCORS(handler)
		}
		handler = middleware.WithRequestLogging(r.Pattern, handler)
		mux.Handle(r.Pattern, handler)
	}
}

type handlers struct {
	workspace    *handlerworkspace.Handler
	presentation *presentation.Handler
	graph        *handlergraph.Handler
	artifacts    *handlerartifacts.Handler
	chat         *handlerchat.Handler
}

func newHandlers(sqlDB *sql.DB) handlers {
	if sqlDB == nil {
		return handlers{}
	}

	wsStore := store.NewWorkspaceStore(sqlDB)
	evStore := store.NewEventStore(sqlDB)
	syncStore := store.NewSyncStore(sqlDB)
	mismatchStore := store.NewMismatchStore(sqlDB)
	entityStore := store.NewEntityStore(sqlDB)
	auditStore := store.NewAuditStore(sqlDB)
	uiStateStore := store.NewWorkspaceUIStateStore(sqlDB)

	embCache := aiworker.NewEmbeddingCache("storage/embeddings")
	aiClient := aiworker.New(aiworker.WithEmbeddingCache(embCache))
	parsedWriter := normalization.NewDocumentWriter("storage/parsed")
	tplExec := execution.TemplateExecutor{PromptsDir: "prompts"}

	shared.SetPersistentIngestService(shared.NewPersistentIngestService(
		wsStore, evStore, entityStore, mismatchStore, syncStore, auditStore,
		shared.WithPersistentParsedWriter(parsedWriter),
		shared.WithPersistentSemanticMatcher(identity.WorkerMatcher{Embedder: aiClient}),
		shared.WithPersistentRelationshipAssistant(relationshipAssistantFromEnv()),
		shared.WithPersistentGraphVerifier(graphVerifierFromEnv(evStore, entityStore)),
	))

	syncWorker := internalsync.NewWorker(wsStore, syncStore, evStore)
	go syncWorker.Run(context.Background(), 15*time.Minute)

	return handlers{
		workspace: handlerworkspace.NewHandler(wsStore, evStore, entityStore, mismatchStore, syncStore).
			WithUIStateRepository(uiStateStore).
			WithAuditRepository(auditStore).
			WithLocalArtifactDirs("storage/parsed", "storage/snapshots").
			WithCodexChatSessionDir("storage/codex-chat-sessions"),
		presentation: presentation.NewHandler(
			wsStore, evStore, mismatchStore, entityStore, syncStore,
			presentation.WithParsedWriter(parsedWriter),
			presentation.WithSemanticMatcher(identity.WorkerMatcher{Embedder: aiClient}),
			presentation.WithExecutor(tplExec),
			presentation.WithAuditRepository(auditStore),
		),
		graph:     handlergraph.NewHandler(wsStore, entityStore),
		artifacts: handlerartifacts.NewHandler(wsStore, evStore),
		chat:      handlerchat.NewHandler(internalchat.NewServiceWithLiveAnswerer(wsStore, evStore, syncStore, internalchat.NewCodexAnswerer())),
	}
}

func relationshipAssistantFromEnv() relationship.Assistant {
	if strings.ToLower(strings.TrimSpace(os.Getenv("CONTEXTOS_AI_RELATIONSHIPS"))) != "codex" {
		return nil
	}
	return relationship.NewCachedAssistant("storage/relationship-cache", relationship.NewCodexAssistant())
}

func graphVerifierFromEnv(events repository.EventRepository, entities repository.EntityRepository) *graphverify.Service {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("CONTEXTOS_GRAPH_VERIFIER")))
	if mode == "" {
		return nil
	}
	switch mode {
	case "data_analytics", "data-analytics":
	case "codex", "codex_cli":
	default:
		return nil
	}
	return &graphverify.Service{
		Events:    events,
		Entities:  entities,
		Assistant: graphverify.NewCodexAssistant(),
		Limit:     80,
	}
}
