// Package codex provides HTTP handlers for the /codex/* routes.
package codex

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/response"
)

// codexPluginStatus represents a single installed plugin.
type codexPluginStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
}

// Status handles GET /codex/status.
// It runs `codex login status` (fast, no network call) and
// `codex plugin list` to report the installed plugin set.
//
// @Summary      Codex CLI status
// @Description  Returns Codex CLI version, login status, and installed plugin list.
// @Tags         codex
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /codex/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	version := codexVersion()
	installed := version != ""
	loggedIn, account := codexLoginStatus()
	plugins := codexPlugins()

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"installed": installed,
		"version":   version,
		"logged_in": loggedIn,
		"account":   account,
		"plugins":   plugins,
	})
}

// Login handles POST /codex/login.
// It runs `codex login --device-auth` and streams its output as SSE log events
// so the browser can display the device-auth URL for the user to open.
// The stream ends with a "result" event once the process exits.
//
// @Summary      Trigger Codex device login
// @Description  Runs `codex login --device-auth` and streams log lines as SSE events.
// @Tags         codex
// @Produce      text/event-stream
// @Success      200
// @Failure      405  {object}  map[string]string
// @Router       /codex/login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	f, ok := shared.SSEHeaders(w)
	if !ok {
		return
	}
	sw := shared.NewSSEWriter(w, f)

	binary := resolveCodexBin()
	cmd := exec.Command(binary, "login", "--device-auth") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = sw
	cmd.Stderr = sw

	if err := cmd.Start(); err != nil {
		sw.Event("error", err.Error())
		return
	}

	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	ctx := r.Context()
	select {
	case err := <-doneCh:
		if err != nil {
			sw.Event("error", err.Error())
		} else {
			sw.Event("result", "ok")
		}
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh
		sw.Event("error", "cancelled")
	case <-time.After(3 * time.Minute):
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh
		sw.Event("error", "login timed out")
	}
}

// PluginReauth handles POST /codex/plugin-reauth?plugin=github|atlassian-rovo|slack.
// It removes the plugin and re-adds it, which triggers a fresh OAuth consent
// flow so the user can connect a different platform account.
// Progress is streamed as SSE log events.
//
// @Summary      Re-authenticate a Codex plugin
// @Description  Removes then re-adds the named plugin to trigger a fresh OAuth flow.
// @Tags         codex
// @Produce      text/event-stream
// @Param        plugin  query  string  true  "Plugin short name: github, atlassian-rovo, jira, or slack"
// @Success      200
// @Failure      400  {object}  map[string]string
// @Failure      405  {object}  map[string]string
// @Router       /codex/plugin-reauth [post]
func PluginReauth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	plugin := strings.TrimSpace(r.URL.Query().Get("plugin"))
	var fullName string
	switch plugin {
	case "github":
		fullName = "github@openai-curated"
	case "atlassian-rovo", "jira":
		fullName = "atlassian-rovo@openai-curated"
	case "slack":
		fullName = "slack@openai-curated"
	default:
		response.WriteError(w, http.StatusBadRequest, "invalid_plugin",
			"plugin must be github, atlassian-rovo, jira, or slack")
		return
	}

	f, ok := shared.SSEHeaders(w)
	if !ok {
		return
	}
	sw := shared.NewSSEWriter(w, f)

	binary := resolveCodexBin()

	// Step 1: remove the plugin.
	sw.Log(fmt.Sprintf("Removing %s...", fullName))
	if err := runCodexSSE(r.Context(), sw, binary, "plugin", "remove", fullName); err != nil {
		sw.Log(fmt.Sprintf("(remove skipped: %s)", err.Error()))
	}

	// Step 2: re-add — this triggers the OAuth consent flow.
	sw.Log(fmt.Sprintf("Re-adding %s - follow the auth prompt...", fullName))
	if err := runCodexSSE(r.Context(), sw, binary, "plugin", "add", fullName); err != nil {
		sw.Event("error", err.Error())
		return
	}

	sw.Event("result", "ok")
}

// codexVersion returns the Codex CLI version string, or empty if not installed.
func codexVersion() string {
	out, err := runCodexInfo("--version")
	if err != nil {
		return ""
	}
	// output: "codex-cli 0.134.0"
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return strings.TrimSpace(out)
}

// codexLoginStatus runs `codex login status` and returns (loggedIn, accountDescription).
func codexLoginStatus() (bool, string) {
	out, err := runCodexInfo("login", "status")
	text := strings.TrimSpace(out)
	if err != nil || strings.HasPrefix(strings.ToLower(text), "not logged") {
		return false, ""
	}
	return true, text
}

// codexPlugins parses `codex plugin list` and returns supported plugin status.
func codexPlugins() []codexPluginStatus {
	out, err := runCodexInfo("plugin", "list")
	plugins := []codexPluginStatus{
		{Name: "github@openai-curated"},
		{Name: "atlassian-rovo@openai-curated"},
		{Name: "slack@openai-curated"},
	}
	if err != nil {
		return plugins
	}
	for i, p := range plugins {
		for _, line := range strings.Split(out, "\n") {
			if !strings.HasPrefix(line, p.Name) {
				continue
			}
			lower := strings.ToLower(line)
			plugins[i].Installed = strings.Contains(lower, "installed")
			plugins[i].Enabled = strings.Contains(lower, "enabled")
		}
	}
	return plugins
}

// resolveCodexBin returns the absolute path to the codex executable,
// falling back to known nvm installation directories when the binary is not
// on the process PATH (common in dev containers started by the API server).
func resolveCodexBin() string {
	if p, err := exec.LookPath("codex"); err == nil {
		return p
	}
	home := os.Getenv("HOME")
	candidates := []string{
		"/home/codespace/nvm/current/bin/codex",
		home + "/nvm/current/bin/codex",
		"/usr/local/share/nvm/current/bin/codex",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "codex"
}

// runCodexInfo runs codex with the given args, capturing combined output.
// It uses a short timeout so the status endpoint stays fast.
func runCodexInfo(args ...string) (string, error) {
	binary := resolveCodexBin()
	cmd := exec.Command(binary, args...) //nolint:gosec
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return buf.String(), err
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return buf.String(), nil
	}
}

// runCodexSSE runs a codex sub-command, streaming output to sw, with a 3-minute timeout.
func runCodexSSE(ctx context.Context, sw *shared.SSEWriter, binary string, args ...string) error {
	cmd := exec.Command(binary, args...) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = sw
	cmd.Stderr = sw

	if err := cmd.Start(); err != nil {
		return err
	}
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	select {
	case err := <-doneCh:
		return err
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh
		return ctx.Err()
	case <-time.After(3 * time.Minute):
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh
		return fmt.Errorf("timed out")
	}
}
