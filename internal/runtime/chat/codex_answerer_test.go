package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestLivePromptRequestsSourceSeparatedProvenance verifies live answers are prompted to preserve per-source activity detail.
func TestLivePromptRequestsSourceSeparatedProvenance(t *testing.T) {
	prompt := livePrompt("GitHub", "context-os/app#1", "what changed?", "zh")

	for _, want := range []string{
		"Structure the final answer by source",
		"exact provenance fields",
		"separate items",
		"Response language: Simplified Chinese",
		"Answer in the response language above",
		"context-os/app#1",
		"what changed?",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("livePrompt() missing %q in %q", want, prompt)
		}
	}
}

// TestCodexAnswererStartsAndResumesConnectorSession verifies live chat stores a connector-scoped session and resumes it on the second turn.
func TestCodexAnswererStartsAndResumesConnectorSession(t *testing.T) {
	command, logPath, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", logPath)
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)

	first, err := answerer.Answer(context.Background(), liveTestQuery("workspace-1"))
	if err != nil {
		t.Fatalf("Answer() first error = %v", err)
	}
	second, err := answerer.Answer(context.Background(), liveTestQuery("workspace-1"))
	if err != nil {
		t.Fatalf("Answer() second error = %v", err)
	}

	if first != `{"answer":"new"}` {
		t.Fatalf("first answer = %q, want new JSON answer", first)
	}
	if second != `{"answer":"resumed"}` {
		t.Fatalf("second answer = %q, want resumed JSON answer", second)
	}
	sessionID, err := answerer.sessions.Load(codexSessionKey("workspace-1", "github"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if sessionID != "session-1" {
		t.Fatalf("sessionID = %q, want session-1", sessionID)
	}
	log := readText(t, logPath)
	if !strings.Contains(log, "new") {
		t.Fatalf("log missing new exec: %q", log)
	}
	if !strings.Contains(log, "resume session-1") {
		t.Fatalf("log missing resume session: %q", log)
	}
	if strings.Count(log, "stdin:") != 2 {
		t.Fatalf("log = %q, want stdin prompt for new and resumed sessions", log)
	}
	if !strings.Contains(log, "what changed?") {
		t.Fatalf("log = %q, want prompt read from stdin", log)
	}
}

// TestCodexAnswererAllowsMissingSessionMeta verifies a live answer can succeed without reusable Codex session metadata.
func TestCodexAnswererAllowsMissingSessionMeta(t *testing.T) {
	command, logPath, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", logPath)
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)
	t.Setenv("FAKE_CODEX_NO_SESSION_META", "1")

	answer, err := answerer.Answer(context.Background(), liveTestQuery("workspace-1"))
	if err != nil {
		t.Fatalf("Answer() error = %v", err)
	}

	if answer != `{"answer":"new"}` {
		t.Fatalf("answer = %q, want new JSON answer", answer)
	}
	sessionID, err := answerer.sessions.Load(codexSessionKey("workspace-1", "github"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if sessionID != "" {
		t.Fatalf("sessionID = %q, want no saved session", sessionID)
	}
	if log := readText(t, logPath); !strings.Contains(log, "new") {
		t.Fatalf("log = %q, want completed new exec", log)
	}
}

// TestCodexAnswererSeparatesWorkspaceSessions verifies each workspace stores and resumes its own Codex session ID.
func TestCodexAnswererSeparatesWorkspaceSessions(t *testing.T) {
	command, logPath, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", logPath)
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)

	if _, err := answerer.Answer(context.Background(), liveTestQuery("workspace-a")); err != nil {
		t.Fatalf("Answer() workspace-a error = %v", err)
	}
	if _, err := answerer.Answer(context.Background(), liveTestQuery("workspace-b")); err != nil {
		t.Fatalf("Answer() workspace-b error = %v", err)
	}

	first, err := answerer.sessions.Load(codexSessionKey("workspace-a", "github"))
	if err != nil {
		t.Fatalf("Load(workspace-a) error = %v", err)
	}
	second, err := answerer.sessions.Load(codexSessionKey("workspace-b", "github"))
	if err != nil {
		t.Fatalf("Load(workspace-b) error = %v", err)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("session IDs = %q and %q, want distinct non-empty values", first, second)
	}
}

// TestCodexAnswererSeparatesConnectorSessions verifies different live connectors keep independent Codex sessions.
func TestCodexAnswererSeparatesConnectorSessions(t *testing.T) {
	command, _, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", filepath.Join(t.TempDir(), "codex.log"))
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)

	if _, err := answerer.Answer(context.Background(), liveTestConnectorQuery("workspace-1", "github")); err != nil {
		t.Fatalf("Answer() github error = %v", err)
	}
	if _, err := answerer.Answer(context.Background(), liveTestConnectorQuery("workspace-1", "slack")); err != nil {
		t.Fatalf("Answer() slack error = %v", err)
	}

	githubSession, err := answerer.sessions.Load(codexSessionKey("workspace-1", "github"))
	if err != nil {
		t.Fatalf("Load(github) error = %v", err)
	}
	slackSession, err := answerer.sessions.Load(codexSessionKey("workspace-1", "slack"))
	if err != nil {
		t.Fatalf("Load(slack) error = %v", err)
	}
	if githubSession == "" || slackSession == "" || githubSession == slackSession {
		t.Fatalf("session IDs = %q and %q, want distinct connector sessions", githubSession, slackSession)
	}
}

// TestCodexAnswererResetSessionDeletesConnectorSessions verifies reset removes legacy and connector-scoped workspace sessions.
func TestCodexAnswererResetSessionDeletesConnectorSessions(t *testing.T) {
	answerer := CodexAnswerer{command: "codex", sessions: NewCodexSessionStore(t.TempDir())}
	if err := answerer.sessions.Save("workspace-1", "legacy-session"); err != nil {
		t.Fatalf("Save(legacy) error = %v", err)
	}
	if err := answerer.sessions.Save(codexSessionKey("workspace-1", "github"), "github-session"); err != nil {
		t.Fatalf("Save(github) error = %v", err)
	}
	if err := answerer.sessions.Save(codexSessionKey("workspace-1", "slack"), "slack-session"); err != nil {
		t.Fatalf("Save(slack) error = %v", err)
	}
	if err := answerer.sessions.Save(codexSessionKey("workspace-2", "github"), "other-session"); err != nil {
		t.Fatalf("Save(other workspace) error = %v", err)
	}

	if err := answerer.ResetSession(context.Background(), "workspace-1"); err != nil {
		t.Fatalf("ResetSession() error = %v", err)
	}

	for _, key := range []string{"workspace-1", codexSessionKey("workspace-1", "github"), codexSessionKey("workspace-1", "slack")} {
		sessionID, err := answerer.sessions.Load(key)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", key, err)
		}
		if sessionID != "" {
			t.Fatalf("Load(%q) = %q, want deleted session", key, sessionID)
		}
	}
	otherSession, err := answerer.sessions.Load(codexSessionKey("workspace-2", "github"))
	if err != nil {
		t.Fatalf("Load(other workspace) error = %v", err)
	}
	if otherSession != "other-session" {
		t.Fatalf("other workspace session = %q, want preserved session", otherSession)
	}
}

// TestCodexAnswererRefreshesUnavailableResumeSession verifies a missing stored connector session starts a fresh session.
func TestCodexAnswererRefreshesUnavailableResumeSession(t *testing.T) {
	command, logPath, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", logPath)
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)
	t.Setenv("FAKE_CODEX_RESUME_FAIL", "1")
	if err := answerer.sessions.Save(codexSessionKey("workspace-1", "github"), "old-session"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	answer, err := answerer.Answer(context.Background(), liveTestQuery("workspace-1"))
	if err != nil {
		t.Fatalf("Answer() error = %v", err)
	}

	if answer != `{"answer":"new"}` {
		t.Fatalf("answer = %q, want fresh answer", answer)
	}
	sessionID, err := answerer.sessions.Load(codexSessionKey("workspace-1", "github"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if sessionID != "session-1" {
		t.Fatalf("sessionID = %q, want refreshed session-1", sessionID)
	}
	log := readText(t, logPath)
	if !strings.Contains(log, "resume old-session") || !strings.Contains(log, "new") {
		t.Fatalf("log = %q, want failed resume then new exec", log)
	}
}

// TestCodexAnswererSerializesSameConnectorLiveCalls verifies concurrent turns for one workspace connector do not race the stored session.
func TestCodexAnswererSerializesSameConnectorLiveCalls(t *testing.T) {
	command, logPath, counterPath := fakeCodexCommand(t)
	answerer := CodexAnswerer{command: command, sessions: NewCodexSessionStore(t.TempDir())}
	t.Setenv("FAKE_CODEX_LOG", logPath)
	t.Setenv("FAKE_CODEX_COUNTER", counterPath)
	t.Setenv("FAKE_CODEX_SLEEP", "0.05")

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := answerer.Answer(context.Background(), liveTestQuery("workspace-1"))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Answer() concurrent error = %v", err)
		}
	}

	log := strings.Fields(readText(t, logPath))
	joined := strings.Join(log, " ")
	if strings.Count(joined, "new") != 1 {
		t.Fatalf("log = %q, want one new exec", joined)
	}
	if !strings.Contains(joined, "resume session-1") {
		t.Fatalf("log = %q, want second call to resume session-1", joined)
	}
}

// TestParseSessionMetaID verifies Codex JSONL session_meta events provide the persisted session ID.
func TestParseSessionMetaID(t *testing.T) {
	log := "{\"type\":\"other\"}\n{\"type\":\"session_meta\",\"payload\":{\"id\":\"abc-123\"}}\n"

	if got := parseSessionMetaID(log); got != "abc-123" {
		t.Fatalf("parseSessionMetaID() = %q, want abc-123", got)
	}
}

// TestProgressBufferSummarizesCodexJSONEvents verifies live chat streams readable progress for Codex JSONL events.
func TestProgressBufferSummarizesCodexJSONEvents(t *testing.T) {
	lines := []string{}
	progress := &progressBuffer{progress: func(line string) {
		lines = append(lines, line)
	}}

	if _, err := progress.Write([]byte("{\"type\":\"tool_call\",\"payload\":{\"tool_name\":\"github.search_issues\"}}\n{\"type\":\"session_meta\",\"payload\":{\"id\":\"abc\"}}\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	progress.Flush()

	if len(lines) != 2 {
		t.Fatalf("progress lines = %#v, want 2 lines", lines)
	}
	if lines[0] != "• Codex event: tool_call — github.search_issues" {
		t.Fatalf("first progress line = %q, want tool call summary", lines[0])
	}
	if lines[1] != "• Codex session metadata received." {
		t.Fatalf("second progress line = %q, want session metadata summary", lines[1])
	}
}

func liveTestQuery(workspaceID string) LiveQuery {
	return liveTestConnectorQuery(workspaceID, "github")
}

func liveTestConnectorQuery(workspaceID, connector string) LiveQuery {
	return LiveQuery{
		WorkspaceID:      workspaceID,
		Connector:        connector,
		SourceURI:        "owner/repo",
		Message:          "what changed?",
		ResponseLanguage: "en",
	}
}

func fakeCodexCommand(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	command := filepath.Join(dir, "codex")
	logPath := filepath.Join(dir, "codex.log")
	counterPath := filepath.Join(dir, "counter")
	script := `#!/bin/sh
log="$FAKE_CODEX_LOG"
counter="$FAKE_CODEX_COUNTER"
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
  fi
  prev="$arg"
done
stdin=$(cat)
if [ -n "$stdin" ]; then
  printf 'stdin:%s\n' "$stdin" >> "$log"
fi
if [ -n "$FAKE_CODEX_SLEEP" ]; then
  sleep "$FAKE_CODEX_SLEEP"
fi
if [ "$2" = "resume" ]; then
  echo "resume $6" >> "$log"
  if [ "$FAKE_CODEX_RESUME_FAIL" = "1" ]; then
    echo "session not found" >&2
    exit 1
  fi
  printf '{"answer":"resumed"}' > "$out"
  exit 0
fi
n=0
if [ -f "$counter" ]; then
  n=$(cat "$counter")
fi
n=$((n + 1))
echo "$n" > "$counter"
sid="session-$n"
echo "new" >> "$log"
if [ "$FAKE_CODEX_NO_SESSION_META" != "1" ]; then
  echo "{\"type\":\"session_meta\",\"payload\":{\"id\":\"$sid\"}}"
fi
printf '{"answer":"new"}' > "$out"
`
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return command, logPath, counterPath
}

func readText(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return string(body)
}
