// Package codex provides HTTP handlers for the /codex/* routes.
package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/response"
	"context-os/internal/codexio"
)

// codexPluginStatus represents a single installed plugin.
type codexPluginStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
}

type sourceCandidate struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	URI       string `json:"uri"`
	Kind      string `json:"kind"`
	Connector string `json:"connector"`
}

type sourceDiscoveryResponse struct {
	Connector string            `json:"connector"`
	Provider  string            `json:"provider"`
	Sources   []sourceCandidate `json:"sources"`
}

const sourceDiscoveryTimeout = 5 * time.Minute

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

// Sources handles GET /codex/sources?connector=<codex-backed connector>.
// It uses the installed Codex plugin for the requested connector so discovery
// follows the same authenticated Codex account/plugin path as ingest.
//
// @Summary      List Codex-accessible sources
// @Description  Uses Codex plugins to list readable sources for github, jira, slack, notion, googledrive, or sharepoint.
// @Tags         codex
// @Produce      json
// @Param        connector  query     string  true  "Connector: github, jira, slack, notion, googledrive, or sharepoint"
// @Success      200        {object}  sourceDiscoveryResponse
// @Failure      400        {object}  map[string]string
// @Failure      405        {object}  map[string]string
// @Failure      502        {object}  map[string]string
// @Router       /codex/sources [get]
func Sources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	connector := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("connector")))
	if !supportsSourceDiscovery(connector) {
		response.WriteError(w, http.StatusBadRequest, "invalid_connector", "connector must be github, jira, slack, notion, googledrive, or sharepoint")
		return
	}

	started := time.Now()
	log.Printf("codex sources: connector=%s start", connector)
	sources, err := discoverSourcesWithCodex(r.Context(), connector)
	if err != nil {
		log.Printf("codex sources: connector=%s error after %s: %v", connector, time.Since(started).Round(time.Millisecond), err)
		response.WriteError(w, http.StatusBadGateway, "codex_discovery_failed", err.Error())
		return
	}
	log.Printf("codex sources: connector=%s done count=%d duration=%s", connector, len(sources), time.Since(started).Round(time.Millisecond))

	response.WriteJSON(w, http.StatusOK, sourceDiscoveryResponse{
		Connector: connector,
		Provider:  "codex",
		Sources:   sources,
	})
}

func supportsSourceDiscovery(connector string) bool {
	switch connector {
	case "github", "jira", "slack", "notion", "googledrive", "sharepoint":
		return true
	default:
		return false
	}
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

	// Step 2: re-add with BROWSER=echo so the OAuth URL is printed into the
	// SSE log instead of silently opening a browser on the server.
	// The user clicks the URL that appears in the connector log to complete auth.
	sw.Log(fmt.Sprintf("Re-adding %s — an OAuth URL will appear below, open it to complete auth...", fullName))
	if err := runCodexSSEEnv(r.Context(), sw, binary, []string{"BROWSER=echo"}, "plugin", "add", fullName); err != nil {
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
	text := cleanCodexInfoOutput(out)
	if err != nil || strings.HasPrefix(strings.ToLower(text), "not logged") {
		return false, ""
	}
	return true, text
}

func cleanCodexInfoOutput(out string) string {
	lines := strings.Split(out, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(strings.ToLower(trimmed), "warning:") {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(clean, "\n")
}

// codexPlugins parses `codex plugin list` and returns supported plugin status.
func codexPlugins() []codexPluginStatus {
	out, err := runCodexInfo("plugin", "list")
	plugins := []codexPluginStatus{
		{Name: "github@openai-curated"},
		{Name: "atlassian-rovo@openai-curated"},
		{Name: "slack@openai-curated"},
		{Name: "google-drive@openai-curated"},
		{Name: "notion@openai-curated"},
		{Name: "sharepoint@openai-curated"},
	}
	if err != nil {
		return plugins
	}
	// codex plugin list output wraps long lines across multiple rows, so we
	// scan the full output for each plugin slug rather than matching a single line.
	lower := strings.ToLower(out)
	for i, p := range plugins {
		slug := strings.ToLower(p.Name)
		if !strings.Contains(lower, slug) {
			continue
		}
		// Find the segment of text near the slug and check for status words.
		idx := strings.Index(lower, slug)
		// Read up to 120 chars after the slug to capture the status column.
		end := idx + len(slug) + 120
		if end > len(lower) {
			end = len(lower)
		}
		segment := lower[idx:end]
		plugins[i].Installed = strings.Contains(segment, "installed")
		plugins[i].Enabled = strings.Contains(segment, "enabled")
	}
	return plugins
}

func discoverSourcesWithCodex(ctx context.Context, connector string) ([]sourceCandidate, error) {
	prompt := sourceDiscoveryPrompt(connector)
	if prompt == "" {
		return nil, fmt.Errorf("unsupported connector %q", connector)
	}

	out, err := runCodexJSON(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var body struct {
		Sources []sourceCandidate `json:"sources"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(out)), &body); err != nil {
		return nil, fmt.Errorf("could not parse Codex source list JSON: %w", err)
	}

	sources := make([]sourceCandidate, 0, len(body.Sources))
	seen := map[string]bool{}
	for _, source := range body.Sources {
		source.Connector = connector
		source.URI = strings.TrimSpace(source.URI)
		source.Label = strings.TrimSpace(source.Label)
		source.ID = strings.TrimSpace(source.ID)
		source.Kind = strings.TrimSpace(source.Kind)
		if source.URI == "" {
			continue
		}
		if source.Label == "" {
			source.Label = source.URI
		}
		if source.ID == "" {
			source.ID = source.URI
		}
		if source.Kind == "" {
			source.Kind = connector
		}
		if seen[source.URI] {
			continue
		}
		seen[source.URI] = true
		sources = append(sources, source)
	}
	return sources, nil
}

func sourceDiscoveryPrompt(connector string) string {
	switch connector {
	case "github":
		return `Use the GitHub Codex plugin to list repositories available to the connected GitHub account. If plugin repository listing is unavailable and the GitHub CLI is already authenticated, you may use read-only gh commands such as gh repo list as a fallback. Do not modify GitHub. Return only compact JSON in this exact shape: {"sources":[{"id":"owner/repo","label":"owner/repo","uri":"owner/repo","kind":"repository"}]}. Include at most 100 repositories.`
	case "jira":
		return `Use the Atlassian Rovo Codex plugin to list Jira projects available to the connected Atlassian account. First call Atlassian Rovo's accessible Atlassian resources tool and use only returned cloudId/url values; do not infer or guess a Jira site, Cloud ID, or tenant. Use Atlassian Rovo's Jira JQL issue search tool first, not generic Rovo workspace search; generic Rovo search can be blocked even when Jira JQL works. Derive project keys from accessible Jira issues using a read-only JQL query such as ORDER BY updated DESC against each accessible Jira cloudId, and do not modify Atlassian or Jira. Return only compact JSON in this exact shape: {"sources":[{"id":"ABC","label":"ABC — Project name","uri":"https://example.atlassian.net/jira/software/c/projects/ABC","kind":"project"}]}. Prefer project URLs for uri when available; otherwise use the project key. Include at most 100 projects.`
	case "slack":
		return `Use the Slack Codex plugin to list channels available to the connected Slack account. Do not modify Slack. Return only compact JSON in this exact shape: {"sources":[{"id":"C123","label":"#channel-name","uri":"#channel-name","kind":"channel"}]}. Include at most 100 public or accessible channels.`
	case "notion":
		return `Use the Notion Codex plugin to list accessible top-level pages and databases from the connected Notion workspace. Do not modify Notion. Return only compact JSON in this exact shape: {"sources":[{"id":"page-or-database-id","label":"Page or database title","uri":"https://notion.so/page-or-database-id","kind":"page"}]}. Use kind "page" or "database". Include at most 100 sources.`
	case "googledrive":
		return `Use the Google Drive Codex plugin to list accessible Google Drive folders and recently relevant Docs, Sheets, or Slides from the connected account. Do not modify Google Drive. Return only compact JSON in this exact shape: {"sources":[{"id":"drive-id","label":"Folder or document name","uri":"https://drive.google.com/drive/folders/id","kind":"folder"}]}. Use kind "folder", "doc", "sheet", or "slide". Include at most 100 sources.`
	case "sharepoint":
		return `Use the SharePoint Codex plugin to list accessible SharePoint sites, document libraries, folders, or OneDrive locations from the connected Microsoft account. Do not modify SharePoint. Return only compact JSON in this exact shape: {"sources":[{"id":"site-or-item-id","label":"Site or folder name","uri":"https://tenant.sharepoint.com/sites/project","kind":"site"}]}. Use kind "site", "library", "folder", or "file". Include at most 100 sources.`
	default:
		return ""
	}
}

func runCodexJSON(ctx context.Context, prompt string) (string, error) {
	binary := resolveCodexBin()
	cmdCtx, cancel := context.WithTimeout(ctx, sourceDiscoveryTimeout)
	defer cancel()

	out, err := os.CreateTemp("", "contextos-codex-sources-*.json")
	if err != nil {
		return "", err
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(outPath) }()

	log.Printf("codex sources: running %s exec for source discovery", binary)
	started := time.Now()
	cmd := exec.Command(binary, "exec", "--sandbox", "read-only", "--ephemeral", "--color", "never", "-o", outPath, prompt) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	buf := codexio.NewBoundedBuffer(codexio.DefaultLogLimit)
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("codex sources: codex exec failed after %s", time.Since(started).Round(time.Millisecond))
			return "", fmt.Errorf("codex discovery failed: %s", strings.TrimSpace(buf.String()))
		}
		log.Printf("codex sources: codex exec completed in %s", time.Since(started).Round(time.Millisecond))
	case <-cmdCtx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done
		log.Printf("codex sources: codex exec timed out after %s", time.Since(started).Round(time.Millisecond))
		return "", fmt.Errorf("codex discovery timed out after %s", sourceDiscoveryTimeout)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		text = strings.TrimSpace(buf.String())
	}
	if text == "" {
		return "", fmt.Errorf("codex returned no source list")
	}
	return text, nil
}

func extractJSONObject(text string) string {
	trimmed := strings.TrimSpace(text)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}
	return trimmed
}

// resolveCodexBin returns the absolute path to the codex executable.
// CODEX_BIN takes precedence, then PATH, then common user-relative nvm paths.
func resolveCodexBin() string {
	if configured := strings.TrimSpace(os.Getenv("CODEX_BIN")); configured != "" {
		return configured
	}
	if p, err := exec.LookPath("codex"); err == nil {
		return p
	}
	home := os.Getenv("HOME")
	nvmDir := os.Getenv("NVM_DIR")
	candidates := []string{
		home + "/nvm/current/bin/codex",
		home + "/.nvm/current/bin/codex",
		nvmDir + "/current/bin/codex",
		nvmDir + "/versions/node/current/bin/codex",
		"/usr/local/share/nvm/current/bin/codex",
	}
	for _, c := range candidates {
		if strings.TrimSpace(c) == "" || strings.HasPrefix(c, "/current/") || strings.HasPrefix(c, "/versions/") {
			continue
		}
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
	buf := codexio.NewBoundedBuffer(codexio.DefaultLogLimit)
	cmd.Stdout = buf
	cmd.Stderr = buf

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
	return runCodexSSEEnv(ctx, sw, binary, nil, args...)
}

// runCodexSSEEnv is like runCodexSSE but merges extraEnv into the subprocess
// environment before starting. Keys in extraEnv override inherited values.
func runCodexSSEEnv(ctx context.Context, sw *shared.SSEWriter, binary string, extraEnv []string, args ...string) error {
	cmd := exec.Command(binary, args...) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(), extraEnv...)
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
