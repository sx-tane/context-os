package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"context-os/domain/repository"
	internalchat "context-os/internal/runtime/chat"
)

// deleteLocalWorkspaceArtifacts removes workspace-scoped parsed files, graph snapshots, and chat session metadata.
func (h *Handler) deleteLocalWorkspaceArtifacts(workspaceID string) error {
	if workspaceID == "" {
		return nil
	}
	if h.parsedDir != "" {
		if err := os.RemoveAll(filepath.Join(h.parsedDir, workspaceID)); err != nil {
			return fmt.Errorf("delete parsed workspace artifacts: %w", err)
		}
	}
	if h.snapshotDir == "" {
		return h.deleteCodexChatSession(workspaceID)
	}
	if err := removeSnapshotFiles(h.snapshotDir, workspaceID+".json"); err != nil {
		return err
	}
	if err := removeSnapshotFiles(h.snapshotDir, workspaceID+"_*.json"); err != nil {
		return err
	}
	return h.deleteCodexChatSession(workspaceID)
}

// deleteCodexChatSession removes the persisted Codex chat session pointer for one workspace.
func (h *Handler) deleteCodexChatSession(workspaceID string) error {
	if h.sessionDir == "" {
		return nil
	}
	if err := internalchat.NewCodexSessionStore(h.sessionDir).Delete(workspaceID); err != nil {
		return fmt.Errorf("delete codex chat session metadata: %w", err)
	}
	return nil
}

// removeSnapshotFiles deletes snapshot files in a directory matching a glob pattern.
func removeSnapshotFiles(dir, pattern string) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("find snapshot artifacts: %w", err)
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete snapshot artifact %q: %w", match, err)
		}
	}
	return nil
}

// workspaceIDForCleanup returns the stored workspace ID or derives the legacy path-based ID.
func workspaceIDForCleanup(path string, workspace *repository.Workspace) string {
	if workspace != nil && strings.TrimSpace(workspace.ID) != "" {
		return workspace.ID
	}
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.TrimPrefix(id, "_")
	if id == "" {
		return "workspace"
	}
	return id
}
