package workspace

import (
	"time"

	"context-os/domain/repository"
)

const workspaceRequestTimeout = 10 * time.Second

// Handler holds the repository dependencies for workspace HTTP handlers.
type Handler struct {
	workspaces  repository.WorkspaceRepository
	events      repository.EventRepository
	entities    repository.EntityRepository
	mismatches  repository.MismatchRepository
	connSync    repository.SyncRepository
	uiState     repository.WorkspaceUIStateRepository
	audit       repository.AuditRepository
	parsedDir   string
	snapshotDir string
	sessionDir  string
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	entities repository.EntityRepository,
	mismatches repository.MismatchRepository,
	connSync repository.SyncRepository,
) *Handler {
	return &Handler{
		workspaces: workspaces,
		events:     events,
		entities:   entities,
		mismatches: mismatches,
		connSync:   connSync,
	}
}

// WithAuditRepository attaches audit row counts to workspace status.
func (h *Handler) WithAuditRepository(audit repository.AuditRepository) *Handler {
	h.audit = audit
	return h
}

// WithUIStateRepository attaches durable frontend workflow state handlers.
func (h *Handler) WithUIStateRepository(uiState repository.WorkspaceUIStateRepository) *Handler {
	h.uiState = uiState
	return h
}

// WithLocalArtifactDirs configures workspace-scoped local JSON cleanup.
func (h *Handler) WithLocalArtifactDirs(parsedDir, snapshotDir string) *Handler {
	h.parsedDir = parsedDir
	h.snapshotDir = snapshotDir
	return h
}

// WithCodexChatSessionDir configures workspace-scoped Codex chat session metadata cleanup.
func (h *Handler) WithCodexChatSessionDir(sessionDir string) *Handler {
	h.sessionDir = sessionDir
	return h
}
