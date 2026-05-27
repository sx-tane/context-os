package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	codexsource "context-os/internal/source/codex"
	githubsource "context-os/internal/source/github"
)

const githubAPIUserURL = "https://api.github.com/user"

// githubUser is the subset of the GitHub /user response we surface.
type githubUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GithubStatus handles GET /github/status. Returns whether a token is available,
// which source provided it (env/none), and the authenticated GitHub username.
//
// @Summary      GitHub connection status
// @Description  Returns token availability, source (env/none), and the authenticated GitHub login.
// @Tags         github
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /github/status [get]
func GithubStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	connected := token != ""
	src := "none"
	login := ""
	name := ""

	if connected {
		src = "env"
		if u, err := resolveGithubUser(r.Context(), token); err == nil {
			login = u.Login
			name = u.Name
		}
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected": connected,
		"source":    src,
		"login":     login,
		"name":      name,
	})
}

// resolveGithubUser calls GET /user on the GitHub REST API with the given token.
func resolveGithubUser(ctx context.Context, token string) (githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIUserURL, nil)
	if err != nil {
		return githubUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return githubUser{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return githubUser{}, err
	}

	var u githubUser
	if err := json.Unmarshal(body, &u); err != nil {
		return githubUser{}, err
	}
	return u, nil
}

// ingestTimeout caps a direct-API GitHub connector call.
const ingestTimeout = 20 * time.Second

// codexIngestTimeout is longer to accommodate Codex CLI + OpenAI API latency.
const codexIngestTimeout = 120 * time.Second

// GithubIngest handles POST /github/ingest by ingesting a single GitHub artifact
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a GitHub artifact
// @Description  Fetches a GitHub issue, PR, or commit by URI and returns a provenance-rich event.
// @Tags         github
// @Accept       json
// @Produce      json
// @Param        body  body      request.GithubIngest   true  "GitHub ingest request"
// @Success      200   {object}  response.GithubIngest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /github/ingest [post]
func GithubIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.GithubIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	uri := strings.TrimSpace(req.URI)
	if uri == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required")
		return
	}

	metadata := map[string]string{}
	if token := strings.TrimSpace(req.Token); token != "" {
		metadata["github_token"] = token
	}

	ctx, cancel := context.WithTimeout(r.Context(), ingestTimeout)
	defer cancel()

	connector := contracts.MCPSourceConnector(githubsource.NewConnector())
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		cancel()
		ctx, cancel = context.WithTimeout(r.Context(), codexIngestTimeout)
		defer cancel()
		connector = codexsource.NewConnector()
		metadata[codexsource.MetadataPlugin] = codexsource.PluginGitHub
		if tok := strings.TrimSpace(req.Token); tok != "" {
			metadata[codexsource.MetadataTokenOverride] = tok
		}
	}

	ingested, err := connector.Ingest(ctx, contracts.SourceRequest{URI: uri, Metadata: metadata})
	if err != nil {
		response.WriteConnectorError(w, err)
		return
	}
	if len(ingested) == 0 {
		response.WriteError(w, http.StatusInternalServerError, "empty_result", "connector returned no events")
		return
	}

	event := ingested[0]
	caps := connector.Capabilities()
	capStrings := make([]string, len(caps))
	for i, c := range caps {
		capStrings[i] = string(c)
	}

	response.WriteJSON(w, http.StatusOK, response.GithubIngest{
		Connector:    connector.Name(),
		Capabilities: capStrings,
		Event:        event,
		Preview:      event.Content,
		Metadata:     event.Metadata,
	})
}
