package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"context-os/internal/codexio"
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

func (a CodexAnswerer) runCodex(ctx context.Context, args []string, prompt string, log *codexio.BoundedBuffer, progress func(string)) error {
	cmd := exec.Command(a.command, args...) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = strings.NewReader(prompt)
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
	buf      *codexio.BoundedBuffer
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
		if summary := summarizeCodexJSONEvent(line); summary != "" {
			p.progress(summary)
		}
		return
	}
	if strings.HasPrefix(line, "›") || strings.HasPrefix(line, "•") {
		p.progress(line)
		return
	}
	p.progress("• " + line)
}

func summarizeCodexJSONEvent(line string) string {
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return ""
	}
	eventType := cleanProgressText(stringField(event, "type"))
	if eventType == "" {
		return ""
	}
	if eventType == "session_meta" {
		return "• Codex session metadata received."
	}
	detail := codexEventDetail(event)
	if detail == "" {
		if isNoisyCodexLifecycleEvent(eventType) {
			return ""
		}
		return "• Codex: " + readableCodexEventType(eventType)
	}
	return "• Codex: " + readableCodexEventType(eventType) + " — " + detail
}

func codexEventDetail(event map[string]any) string {
	for _, key := range []string{"message", "text", "summary", "status", "name", "tool_name", "call_id"} {
		if value := cleanProgressText(stringField(event, key)); value != "" {
			return truncateProgressText(value)
		}
	}
	payload, _ := event["payload"].(map[string]any)
	for _, key := range []string{"message", "text", "summary", "status", "name", "tool_name", "call_id"} {
		if value := cleanProgressText(stringField(payload, key)); value != "" {
			return truncateProgressText(value)
		}
	}
	if item, _ := payload["item"].(map[string]any); item != nil {
		if value := cleanProgressText(firstNonEmpty(
			stringField(item, "tool_name"),
			stringField(item, "name"),
			stringField(item, "title"),
			stringField(item, "command"),
			stringField(item, "status"),
			stringField(item, "type"),
		)); value != "" {
			return truncateProgressText(value)
		}
		if args, ok := item["arguments"].(map[string]any); ok {
			if value := cleanProgressText(firstNonEmpty(stringField(args, "query"), stringField(args, "q"), stringField(args, "url"))); value != "" {
				return truncateProgressText(value)
			}
		}
	}
	if delta, _ := payload["delta"].(map[string]any); delta != nil {
		if value := cleanProgressText(firstNonEmpty(stringField(delta, "message"), stringField(delta, "text"), stringField(delta, "summary"))); value != "" {
			return truncateProgressText(value)
		}
	}
	return ""
}

func isNoisyCodexLifecycleEvent(eventType string) bool {
	switch eventType {
	case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
		return true
	default:
		return false
	}
}

func readableCodexEventType(eventType string) string {
	switch eventType {
	case "tool_call":
		return "tool call"
	case "agent_message":
		return "message"
	case "reasoning":
		return "reasoning"
	default:
		return strings.ReplaceAll(eventType, "_", " ")
	}
}

func stringField(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64, bool:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}

func cleanProgressText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func truncateProgressText(value string) string {
	const maxProgressDetail = 180
	runes := []rune(value)
	if len(runes) <= maxProgressDetail {
		return value
	}
	return string(runes[:maxProgressDetail-3]) + "..."
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

func codexSessionKey(workspaceID, connector string) string {
	workspaceID = strings.TrimSpace(workspaceID)
	connector = normalizeConnector(connector)
	if connector == "" {
		return workspaceID
	}
	return workspaceID + "::" + connector
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
	jiraRules := ""
	if plugin == "Atlassian Rovo" {
		jiraRules = `
- For Jira questions, use Atlassian Rovo's Jira JQL issue search tool first, not generic Rovo workspace search. Generic Rovo search can be blocked even when Jira JQL works.
- Build a focused JQL query from the question and source. For connector-wide Jira questions, prefer currentUser(), project keys, issue keys, updated/created/due dates, and ORDER BY updated DESC.
- If generic Rovo search returns "app is not installed on this instance", retry through Jira JQL before declaring Jira unavailable.`
	}
	return fmt.Sprintf(`Use the %s Codex plugin to answer this user question from the live connected account.

Source: %s
Question: %s
Response language: %s

Rules:
- Do not modify any external data.
- Answer in the response language above. If the user mixed languages, prefer the language used for the actual question.
- Start with the direct answer to the user's question and the decision or next action when the evidence supports one.
- Prefer exact source facts over general repository or workspace summaries.
- Use only the %s Codex plugin or context it returns. Do not use gh, git remotes, public web search, or other local/public fallbacks.
- If Source is only a connector name such as github, jira, slack, or googledrive, treat it as the connected account scope for that plugin. Do not inspect or cite unrelated public sources outside the connected account.
- Prefer product-specific read tools inside the selected plugin over broad workspace search when both exist.%s
- Include only the strongest provenance needed to support the answer: source names, links, issue or PR numbers, timestamps, authors, or commit hashes when they materially matter.
- Structure evidence by source so each artifact, thread, issue, PR, or document stays traceable without creating a long inventory.
- Return only JSON with this shape: {"answer":"short plain-text summary","answer_sections":[{"source_label":"human source name","connector":"github|jira|slack|googledrive|notion|sharepoint","source_uri":"exact source URI or key","summary":"short summary","facts":["fact"],"open_items":["open item"],"coding_notes":["coding note"],"links":["https://..."],"timestamps":["timestamp"],"confidence":0.0,"status":"optional status"}]}.
- Use one answer_sections item per real source or artifact. Do not create sections from URL path fragments, enum values, generic terms, or prose tokens.
- Return at most 5 answer_sections unless more are required to avoid a misleading answer.
- In each section, include at most 3 facts and explain why that source changes or supports the answer.
- If multiple activities or thread messages say the same thing, merge them into one concise section and keep the best links.
- If the plugin cannot access the source or the requested fact is unavailable, say that clearly.
- Keep answer concise and readable for chat.`, plugin, sourceURI, strings.TrimSpace(message), language, plugin, jiraRules)
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
