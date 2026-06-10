package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"context-os/internal/codexio"
)

// CodexAnswerer answers chat questions by invoking Codex CLI with installed plugins.
type CodexAnswerer struct {
	command  string
	sessions *codexSessionStore
}

// NewCodexAnswerer returns a live answerer backed by the local Codex CLI.
func NewCodexAnswerer() CodexAnswerer {
	return CodexAnswerer{command: resolveCodexCommand(), sessions: NewCodexSessionStore(defaultCodexSessionDir)}
}

// Answer asks a live Codex plugin for source-specific context.
func (a CodexAnswerer) Answer(ctx context.Context, query LiveQuery) (string, error) {
	if a.sessions == nil {
		a.sessions = NewCodexSessionStore(defaultCodexSessionDir)
	}
	plugin := livePlugin(query.Connector)
	if plugin == "" {
		return "", fmt.Errorf("unsupported live connector %q", query.Connector)
	}
	sourceURI := strings.TrimSpace(query.SourceURI)
	if sourceURI == "" {
		return "", errors.New("source_uri is required for live chat")
	}
	workspaceID := strings.TrimSpace(firstNonEmpty(query.WorkspaceID, query.WorkspacePath))
	if workspaceID == "" {
		return "", errors.New("workspace_id is required for live chat")
	}
	sessionKey := codexSessionKey(workspaceID, query.Connector)

	unlock := a.sessions.lockWorkspace(sessionKey)
	defer unlock()
	sessionID, err := a.sessions.Load(sessionKey)
	if err != nil {
		return "", err
	}
	out, err := os.CreateTemp("", "contextos-chat-codex-*.txt")
	if err != nil {
		return "", err
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(outPath) }()

	prompt := livePrompt(plugin, sourceURI, query.Message, query.ResponseLanguage)
	emitProgress(query.Progress, fmt.Sprintf("› Live Codex: %s plugin lookup", plugin))
	emitProgress(query.Progress, fmt.Sprintf("• Source: %s", sourceURI))

	log := codexio.NewBoundedBuffer(codexio.DefaultLogLimit)
	if sessionID != "" {
		emitProgress(query.Progress, "• Resuming Codex CLI chat session.")
		err = a.runCodex(ctx, []string{"exec", "resume", "--json", "-o", outPath, sessionID, "-"}, prompt, log, query.Progress)
		if err != nil && isResumeSessionUnavailable(log.String(), err) {
			emitProgress(query.Progress, "• Stored Codex session is unavailable; starting a fresh chat session.")
			if deleteErr := a.sessions.Delete(sessionKey); deleteErr != nil {
				return "", deleteErr
			}
			sessionID = ""
			log.Reset()
		} else if err != nil {
			return "", err
		}
	}
	if sessionID == "" {
		emitProgress(query.Progress, "• Starting new Codex CLI chat session.")
		if err := a.runCodex(ctx, []string{"exec", "--sandbox", "read-only", "--json", "--color", "never", "-o", outPath, "-"}, prompt, log, query.Progress); err != nil {
			return "", err
		}
		nextSessionID := parseSessionMetaID(log.String())
		if nextSessionID == "" {
			emitProgress(query.Progress, "• Codex CLI did not report a reusable session id; continuing without resume metadata.")
		} else if err := a.sessions.Save(sessionKey, nextSessionID); err != nil {
			return "", err
		}
	}
	emitProgress(query.Progress, "• Codex CLI completed; reading answer.")

	content, err := os.ReadFile(filepath.Clean(outPath))
	if err != nil {
		return "", err
	}
	answer := strings.TrimSpace(string(content))
	if answer == "" {
		answer = strings.TrimSpace(log.String())
	}
	if answer == "" {
		return "", errors.New("codex returned no live chat answer")
	}
	return answer, nil
}

// ResetSession deletes stored Codex chat session metadata for a workspace.
func (a CodexAnswerer) ResetSession(ctx context.Context, workspaceID string) error {
	if a.sessions == nil {
		a.sessions = NewCodexSessionStore(defaultCodexSessionDir)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	unlock := a.sessions.lockWorkspace(workspaceID)
	defer unlock()
	return a.sessions.DeleteWorkspace(workspaceID)
}
