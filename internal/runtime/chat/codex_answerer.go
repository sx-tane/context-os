package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"sync"
	"time"
)

var codexBinCandidates = []string{
	"${HOME}/nvm/current/bin/codex",
	"${HOME}/.nvm/current/bin/codex",
	"${NVM_DIR}/current/bin/codex",
	"${NVM_DIR}/versions/node/current/bin/codex",
	"/usr/local/share/nvm/current/bin/codex",
}

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

	unlock := a.sessions.lockWorkspace(workspaceID)
	defer unlock()
	sessionID, err := a.sessions.Load(workspaceID)
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

	var log bytes.Buffer
	if sessionID != "" {
		emitProgress(query.Progress, "• Resuming Codex CLI chat session.")
		err = a.runCodex(ctx, []string{"exec", "resume", "--json", "-o", outPath, sessionID, prompt}, &log, query.Progress)
		if err != nil && isResumeSessionUnavailable(log.String(), err) {
			emitProgress(query.Progress, "• Stored Codex session is unavailable; starting a fresh chat session.")
			if deleteErr := a.sessions.Delete(workspaceID); deleteErr != nil {
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
		if err := a.runCodex(ctx, []string{"exec", "--sandbox", "read-only", "--json", "--color", "never", "-o", outPath, prompt}, &log, query.Progress); err != nil {
			return "", err
		}
		nextSessionID := parseSessionMetaID(log.String())
		if nextSessionID == "" {
			return "", errors.New("codex did not report a session_meta id")
		}
		if err := a.sessions.Save(workspaceID, nextSessionID); err != nil {
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
	return a.sessions.Delete(workspaceID)
}

func (a CodexAnswerer) runCodex(ctx context.Context, args []string, log *bytes.Buffer, progress func(string)) error {
	cmd := exec.Command(a.command, args...) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	progressLog := &progressBuffer{buf: log, progress: progress}
	cmd.Stdout = progressLog
	cmd.Stderr = progressLog
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		progressLog.Flush()
		if err != nil {
			text := strings.TrimSpace(log.String())
			if text == "" {
				text = err.Error()
			}
			return fmt.Errorf("codex live chat failed: %s", text)
		}
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done
		progressLog.Flush()
		return fmt.Errorf("codex live chat canceled: %w", ctx.Err())
	}
}

type progressBuffer struct {
	buf      *bytes.Buffer
	progress func(string)
	partial  string
}

func (p *progressBuffer) Write(data []byte) (int, error) {
	if p.buf != nil {
		_, _ = p.buf.Write(data)
	}
	text := p.partial + string(data)
	lines := strings.Split(text, "\n")
	p.partial = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		p.emit(line)
	}
	return len(data), nil
}

func (p *progressBuffer) Flush() {
	p.emit(p.partial)
	p.partial = ""
}

func (p *progressBuffer) emit(line string) {
	line = strings.TrimSpace(line)
	if line == "" || p.progress == nil {
		return
	}
	if strings.HasPrefix(line, "{") {
		return
	}
	if strings.HasPrefix(line, "›") || strings.HasPrefix(line, "•") {
		p.progress(line)
		return
	}
	p.progress("• " + line)
}

func resolveCodexCommand() string {
	if configured := strings.TrimSpace(os.Getenv("CODEX_BIN")); configured != "" {
		return configured
	}
	if path, err := exec.LookPath("codex"); err == nil {
		return path
	}
	home := os.Getenv("HOME")
	nvmDir := os.Getenv("NVM_DIR")
	for _, candidate := range codexBinCandidates {
		candidate = strings.ReplaceAll(candidate, "${HOME}", home)
		candidate = strings.ReplaceAll(candidate, "${NVM_DIR}", nvmDir)
		if strings.Contains(candidate, "${") {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "codex"
}

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

func isResumeSessionUnavailable(log string, err error) bool {
	text := strings.ToLower(strings.TrimSpace(log + " " + err.Error()))
	return strings.Contains(text, "missing") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "archived") ||
		strings.Contains(text, "unreadable") ||
		strings.Contains(text, "permission denied")
}

func supportsLiveConnector(connector string) bool {
	return livePlugin(connector) != ""
}

func livePlugin(connector string) string {
	switch normalizeConnector(connector) {
	case "github":
		return "GitHub"
	case "slack":
		return "Slack"
	case "jira":
		return "Atlassian Rovo"
	case "googledrive":
		return "Google Drive"
	case "notion":
		return "Notion"
	case "sharepoint":
		return "SharePoint"
	default:
		return ""
	}
}

func livePrompt(plugin, sourceURI, message, responseLanguage string) string {
	language := normalizeResponseLanguage(responseLanguage)
	return fmt.Sprintf(`Use the %s Codex plugin to answer this user question from the live connected account.

Source: %s
Question: %s
Response language: %s

Rules:
- Do not modify any external data.
- Answer in the response language above. If the user mixed languages, prefer the language used for the actual question.
- Prefer exact source facts over general repository or workspace summaries.
- For GitHub only, if the plugin cannot answer and gh CLI is already authenticated, read-only gh commands are acceptable fallback context.
- Include source names, timestamps, authors, commit hashes, issue or PR numbers, and links when available.
- Structure the final answer by source so each artifact, thread, issue, PR, or document stays separately traceable.
- Return only JSON with this shape: {"answer":"short plain-text summary","answer_sections":[{"source_label":"human source name","connector":"github|jira|slack|googledrive|notion|sharepoint","source_uri":"exact source URI or key","summary":"short summary","facts":["fact"],"open_items":["open item"],"coding_notes":["coding note"],"links":["https://..."],"timestamps":["timestamp"],"confidence":0.0,"status":"optional status"}]}.
- Use one answer_sections item per real source or artifact. Do not create sections from URL path fragments, enum values, generic terms, or prose tokens.
- In each section, include factual summary, exact provenance fields available, and why that source is relevant to the question.
- If multiple activities or thread messages are relevant, keep them as separate items instead of merging them into one vague event.
- If the plugin cannot access the source or the requested fact is unavailable, say that clearly.
- Keep answer concise and readable for chat.`, plugin, sourceURI, strings.TrimSpace(message), language)
}

func normalizeResponseLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "zh", "zh-cn", "cn":
		return "Simplified Chinese"
	case "zh-tw", "zh-hant":
		return "Traditional Chinese"
	case "ja", "jp":
		return "Japanese"
	case "ko", "kr":
		return "Korean"
	default:
		return "English"
	}
}
