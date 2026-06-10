package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const defaultCodexSessionDir = "storage/codex-chat-sessions"

// NewCodexSessionStore returns a file-backed Codex chat session metadata store.
func NewCodexSessionStore(dir string) *codexSessionStore {
	return &codexSessionStore{dir: dir, locks: map[string]*sync.Mutex{}}
}

type codexSessionStore struct {
	dir   string
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

type codexSessionMetadata struct {
	WorkspaceID string    `json:"workspace_id"`
	SessionID   string    `json:"session_id"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Load returns the stored Codex CLI session ID for a workspace connector key, or an empty ID when none exists.
func (s *codexSessionStore) Load(workspaceID string) (string, error) {
	path, err := s.path(workspaceID)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read codex chat session: %w", err)
	}
	var meta codexSessionMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("decode codex chat session: %w", err)
	}
	return strings.TrimSpace(meta.SessionID), nil
}

// Save persists the Codex CLI session ID for a workspace connector key.
func (s *codexSessionStore) Save(workspaceID, sessionID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	sessionID = strings.TrimSpace(sessionID)
	if workspaceID == "" {
		return errors.New("workspace_id is required")
	}
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	path, err := s.path(workspaceID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create codex chat session dir: %w", err)
	}
	body, err := json.MarshalIndent(codexSessionMetadata{
		WorkspaceID: workspaceID,
		SessionID:   sessionID,
		UpdatedAt:   time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode codex chat session: %w", err)
	}
	return os.WriteFile(path, append(body, '\n'), 0o600)
}

// Delete removes the stored session pointer for a workspace connector key.
func (s *codexSessionStore) Delete(workspaceID string) error {
	path, err := s.path(workspaceID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete codex chat session: %w", err)
	}
	return nil
}

// DeleteWorkspace removes legacy and connector-scoped session pointers for one workspace.
func (s *codexSessionStore) DeleteWorkspace(workspaceID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return errors.New("workspace_id is required")
	}
	if err := s.Delete(workspaceID); err != nil {
		return err
	}
	prefix := safeSessionFilename(workspaceID + "::")
	matches, err := filepath.Glob(filepath.Join(s.dir, prefix+"*.json"))
	if err != nil {
		return fmt.Errorf("match codex chat sessions: %w", err)
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete codex chat session: %w", err)
		}
	}
	return nil
}

// lockWorkspace serializes Codex CLI calls that share the same workspace connector session pointer.
func (s *codexSessionStore) lockWorkspace(workspaceID string) func() {
	key := strings.TrimSpace(workspaceID)
	s.mu.Lock()
	lock := s.locks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[key] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

// path returns the metadata file path for a workspace connector session pointer.
func (s *codexSessionStore) path(workspaceID string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return "", errors.New("workspace_id is required")
	}
	if strings.TrimSpace(s.dir) == "" {
		return "", errors.New("codex chat session dir is required")
	}
	return filepath.Join(s.dir, safeSessionFilename(workspaceID)+".json"), nil
}

var safeSessionCharPattern = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// safeSessionFilename converts a workspace connector key into a filesystem-safe bounded filename stem.
func safeSessionFilename(workspaceID string) string {
	name := safeSessionCharPattern.ReplaceAllString(workspaceID, "_")
	name = strings.Trim(name, "_.-")
	if name == "" {
		return "workspace"
	}
	if len(name) > 180 {
		name = name[:180]
	}
	return name
}

// codexSessionKey builds connector-scoped session keys and versions Jira sessions for JQL routing changes.
func codexSessionKey(workspaceID, connector string) string {
	workspaceID = strings.TrimSpace(workspaceID)
	connector = normalizeConnector(connector)
	if connector == "" {
		return workspaceID
	}
	if connector == "jira" {
		return workspaceID + "::jira-jql-v2"
	}
	return workspaceID + "::" + connector
}

// parseSessionMetaID extracts the reusable Codex CLI session ID from JSONL session_meta output.
func parseSessionMetaID(log string) string {
	for _, line := range strings.Split(log, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event struct {
			Type    string `json:"type"`
			Payload struct {
				ID string `json:"id"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "session_meta" && strings.TrimSpace(event.Payload.ID) != "" {
			return strings.TrimSpace(event.Payload.ID)
		}
	}
	return ""
}

// isResumeSessionUnavailable reports whether Codex rejected a stored session pointer as unavailable.
func isResumeSessionUnavailable(log string, err error) bool {
	text := strings.ToLower(strings.TrimSpace(log + " " + err.Error()))
	return strings.Contains(text, "missing") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "archived") ||
		strings.Contains(text, "unreadable") ||
		strings.Contains(text, "permission denied")
}
