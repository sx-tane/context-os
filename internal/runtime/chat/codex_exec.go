package chat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"context-os/internal/codexio"
)

var codexBinCandidates = []string{
	"${HOME}/nvm/current/bin/codex",
	"${HOME}/.nvm/current/bin/codex",
	"${NVM_DIR}/current/bin/codex",
	"${NVM_DIR}/versions/node/current/bin/codex",
	"/usr/local/share/nvm/current/bin/codex",
}

// runCodex executes the Codex CLI with a prompt on stdin and streams stdout/stderr through a progress parser.
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

// resolveCodexCommand selects the Codex CLI path from configuration, PATH, common nvm locations, or a plain fallback name.
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
