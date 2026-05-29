// Package github provides HTTP handlers for the /github/* routes.
package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	codexsource "context-os/internal/source/codex"
	githubsource "context-os/internal/source/github"
)

// Status handles GET /github/status.
// It checks the GITHUB_TOKEN environment variable and probes the GitHub API
// to return the connected account identity.
//
// @Summary      GitHub connection status
// @Description  Returns whether a GitHub token is configured and the authenticated user identity.
// @Tags         github
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /github/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		response.WriteJSON(w, http.StatusOK, map[string]any{
			"connected": false,
			"source":    "env",
		})
		return
	}

	login, name := resolveUser(token)
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected": true,
		"source":    "env",
		"login":     login,
		"name":      name,
	})
}

// Ingest handles POST /github/ingest by ingesting a GitHub artifact
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a GitHub artifact
// @Description  Fetches a GitHub issue, PR, or commit by URI and returns a provenance-rich event.
// @Tags         github
// @Accept       json
// @Produce      json
// @Param        body  body      request.GithubIngest   true  "GitHub ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /github/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.GithubIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	metadata := map[string]string{}
	if token := strings.TrimSpace(req.Token); token != "" {
		metadata["github_token"] = token
	}

	connector := contracts.MCPSourceConnector(githubsource.NewConnector())
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		connector = codexsource.NewConnector()
		metadata[codexsource.MetadataPlugin] = codexsource.PluginGitHub
	}

	shared.WriteSourceIngest(w, r, connector, shared.SourceIngestInput{URI: req.URI, Metadata: metadata})
}

// IngestStream handles POST /github/ingest/stream.
// It streams Codex CLI log lines as SSE "log" events while the process runs,
// then emits a single "result" event with the final ingest payload.
// Only the "codex" provider is accepted; for the token provider use /github/ingest.
//
// @Summary      Stream GitHub Codex ingest
// @Description  Streams Codex CLI progress via SSE, then emits a result event.
// @Tags         github
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.GithubIngest  true  "GitHub ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /github/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginGitHub,
		[]string{"repository", "issues"},
		func(dec *json.Decoder) (request.GithubIngest, error) {
			var req request.GithubIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.GithubIngest) string { return req.URI },
		func(req request.GithubIngest) string { return req.Provider },
		func(req request.GithubIngest) string { return req.Token },
	)
}

// resolveUser calls the GitHub REST API to retrieve the authenticated user's
// login and display name for the given personal access token.
func resolveUser(token string) (login, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", ""
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var payload struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", ""
	}
	return payload.Login, payload.Name
}
