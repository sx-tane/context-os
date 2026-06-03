package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const liveAnswerTimeout = 5 * time.Minute

var codexBinCandidates = []string{
	"${HOME}/nvm/current/bin/codex",
	"${HOME}/.nvm/current/bin/codex",
	"${NVM_DIR}/current/bin/codex",
	"${NVM_DIR}/versions/node/current/bin/codex",
	"/usr/local/share/nvm/current/bin/codex",
}

// CodexAnswerer answers chat questions by invoking Codex CLI with installed plugins.
type CodexAnswerer struct {
	command string
}

// NewCodexAnswerer returns a live answerer backed by the local Codex CLI.
func NewCodexAnswerer() CodexAnswerer {
	return CodexAnswerer{command: resolveCodexCommand()}
}

// Answer asks a live Codex plugin for source-specific context.
func (a CodexAnswerer) Answer(ctx context.Context, query LiveQuery) (string, error) {
	plugin := livePlugin(query.Connector)
	if plugin == "" {
		return "", fmt.Errorf("unsupported live connector %q", query.Connector)
	}
	sourceURI := strings.TrimSpace(query.SourceURI)
	if sourceURI == "" {
		return "", errors.New("source_uri is required for live chat")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, liveAnswerTimeout)
	defer cancel()

	out, err := os.CreateTemp("", "contextos-chat-codex-*.txt")
	if err != nil {
		return "", err
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(outPath) }()

	prompt := livePrompt(plugin, sourceURI, query.Message)
	cmd := exec.Command(a.command, "exec", "--sandbox", "read-only", "--ephemeral", "--color", "never", "-o", outPath, prompt) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var log bytes.Buffer
	progressLog := &progressBuffer{buf: &log, progress: query.Progress}
	emitProgress(query.Progress, fmt.Sprintf("› Live Codex: %s plugin lookup", plugin))
	emitProgress(query.Progress, fmt.Sprintf("• Source: %s", sourceURI))
	emitProgress(query.Progress, "• Starting Codex CLI exec.")
	cmd.Stdout = progressLog
	cmd.Stderr = progressLog
	if err := cmd.Start(); err != nil {
		return "", err
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
			return "", fmt.Errorf("codex live chat failed: %s", text)
		}
	case <-cmdCtx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done
		progressLog.Flush()
		return "", fmt.Errorf("codex live chat timed out after %s", liveAnswerTimeout)
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

func livePrompt(plugin, sourceURI, message string) string {
	return fmt.Sprintf(`Use the %s Codex plugin to answer this user question from the live connected account.

Source: %s
Question: %s

Rules:
- Do not modify any external data.
- Prefer exact source facts over general repository or workspace summaries.
- For GitHub only, if the plugin cannot answer and gh CLI is already authenticated, read-only gh commands are acceptable fallback context.
- Include source names, timestamps, authors, commit hashes, issue or PR numbers, and links when available.
- If the plugin cannot access the source or the requested fact is unavailable, say that clearly.
- Return a concise chat answer, not JSON.`, plugin, sourceURI, strings.TrimSpace(message))
}
