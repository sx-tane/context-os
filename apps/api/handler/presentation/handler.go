package presentation

import (
	"context-os/domain/repository"
	"context-os/internal/stages/execution"
	"context-os/internal/stages/identity"
	"context-os/internal/stages/normalization"
	"time"
)

// findingsTimeout allows Codex-backed presentation analysis to complete while
// still bounding user-triggered findings calls.
const findingsTimeout = 5 * time.Minute
const presentationWriteTimeout = 30 * time.Second

const metadataProductConnector = "product_connector"

// findingsCacheTTL is how fresh persisted mismatches must be to be returned
// directly from DB without re-ingesting.
const findingsCacheTTL = 30 * time.Minute

// Handler holds optional repository dependencies for the presentation endpoints.
// All fields are optional — a nil field disables that capability.
type Handler struct {
	workspaces      repository.WorkspaceRepository
	events          repository.EventRepository
	mismatches      repository.MismatchRepository
	entities        repository.EntityRepository
	syncRepo        repository.SyncRepository
	audit           repository.AuditRepository
	parsedWriter    *normalization.DocumentWriter // persists NormalizedDocuments to storage/parsed/
	semanticMatcher identity.Matcher              // optional Layer-2 semantic identity pass
	executor        execution.CodexExecutor       // optional assistive execution backend
}

// HandlerOption configures optional capabilities on a Handler.
type HandlerOption func(*Handler)

// WithParsedWriter attaches a DocumentWriter that persists parsed documents to disk.
func WithParsedWriter(w *normalization.DocumentWriter) HandlerOption {
	return func(h *Handler) { h.parsedWriter = w }
}

// WithSemanticMatcher attaches an embedding-backed Matcher for Layer-2 identity resolution.
func WithSemanticMatcher(m identity.Matcher) HandlerOption {
	return func(h *Handler) { h.semanticMatcher = m }
}

// WithExecutor overrides the default LocalStubExecutor with a real or template-backed executor.
func WithExecutor(e execution.CodexExecutor) HandlerOption {
	return func(h *Handler) { h.executor = e }
}

// WithAuditRepository attaches audit logging for findings pipeline actions.
func WithAuditRepository(a repository.AuditRepository) HandlerOption {
	return func(h *Handler) { h.audit = a }
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	mismatches repository.MismatchRepository,
	entities repository.EntityRepository,
	syncRepo repository.SyncRepository,
	opts ...HandlerOption,
) *Handler {
	h := &Handler{
		workspaces: workspaces,
		events:     events,
		mismatches: mismatches,
		entities:   entities,
		syncRepo:   syncRepo,
		executor:   execution.LocalStubExecutor{}, // safe default
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
