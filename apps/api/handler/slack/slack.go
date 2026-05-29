// Package slack provides HTTP handlers for the /slack/* routes,
// including OAuth flow helpers and token persistence.
package slack

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	codexsource "context-os/internal/source/codex"
	slacksource "context-os/internal/source/slack"
)

const (
	slackOAuthScopes        = "channels:history,channels:read"
	slackOAuthAuthorizeURL  = "https://slack.com/oauth/v2/authorize"
	slackOAuthTokenURL      = "https://slack.com/api/oauth.v2.access"
	slackDefaultRedirectURI = "http://localhost:8080/slack/callback"
)

// slackStateEntry tracks an outstanding CSRF state token with its creation time.
type slackStateEntry struct {
	createdAt time.Time
}

var slackStateStore sync.Map

// slackStoredToken is persisted to disk after a successful OAuth exchange.
type slackStoredToken struct {
	AccessToken string `json:"access_token"`
	TeamID      string `json:"team_id"`
	TeamName    string `json:"team_name"`
	Scope       string `json:"scope"`
}

// Status handles GET /slack/status. Returns whether a token is available,
// which source provided it (env, oauth, none), the workspace name when known,
// and whether OAuth can be initiated (SLACK_CLIENT_ID + SLACK_CLIENT_SECRET set).
//
// @Summary      Slack connection status
// @Description  Returns token availability, source (env/oauth/none), workspace name, and OAuth readiness.
// @Tags         slack
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /slack/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	connected := false
	source := "none"
	teamName := ""

	if strings.TrimSpace(os.Getenv("SLACK_BOT_TOKEN")) != "" {
		connected = true
		source = "env"
	} else if tok, err := loadToken(); err == nil && tok.AccessToken != "" {
		connected = true
		source = "oauth"
		teamName = tok.TeamName
	}

	oauthAvailable := os.Getenv("SLACK_CLIENT_ID") != "" && os.Getenv("SLACK_CLIENT_SECRET") != ""

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":       connected,
		"source":          source,
		"team_name":       teamName,
		"oauth_available": oauthAvailable,
	})
}

// Connect handles GET /slack/connect. Generates a CSRF state token and
// redirects the browser to Slack's OAuth authorization page.
//
// @Summary      Initiate Slack OAuth
// @Description  Generates a CSRF state token and redirects to Slack's OAuth consent page.
// @Tags         slack
// @Produce      html
// @Success      302  {string}  string  "Redirect to Slack OAuth"
// @Failure      405  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /slack/connect [get]
func Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	clientID := os.Getenv("SLACK_CLIENT_ID")
	if clientID == "" {
		response.WriteError(w, http.StatusServiceUnavailable, "not_configured",
			"SLACK_CLIENT_ID is not set; follow the setup instructions to create a Slack app")
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "state_error", "failed to generate state")
		return
	}
	slackStateStore.Store(state, slackStateEntry{createdAt: time.Now()})
	purgeExpiredOAuthStates()

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("scope", slackOAuthScopes)
	params.Set("redirect_uri", redirectURI())
	params.Set("state", state)

	http.Redirect(w, r, slackOAuthAuthorizeURL+"?"+params.Encode(), http.StatusFound)
}

// Callback handles GET /slack/callback. Verifies the CSRF state, exchanges
// the authorization code for an access token, persists it locally, and returns
// a self-closing HTML page that notifies the opener via postMessage.
//
// @Summary      Slack OAuth callback
// @Description  Receives the OAuth code from Slack, exchanges it for a token, and stores it locally.
// @Tags         slack
// @Produce      html
// @Param        code   query  string  true  "Authorization code from Slack"
// @Param        state  query  string  true  "CSRF state token"
// @Success      200    {string}  string  "HTML confirmation page"
// @Failure      405    {object}  map[string]string
// @Router       /slack/callback [get]
func Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		writeOAuthPage(w, false, "", "Authorization denied: "+html.EscapeString(errParam))
		return
	}

	state := q.Get("state")
	code := q.Get("code")
	if state == "" || code == "" {
		writeOAuthPage(w, false, "", "Missing state or code parameter.")
		return
	}
	if _, ok := slackStateStore.LoadAndDelete(state); !ok {
		writeOAuthPage(w, false, "", "Invalid or expired state. Please try connecting again.")
		return
	}

	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		writeOAuthPage(w, false, "", "SLACK_CLIENT_ID and SLACK_CLIENT_SECRET must both be set.")
		return
	}

	tok, err := exchangeCode(r.Context(), clientID, clientSecret, code)
	if err != nil {
		writeOAuthPage(w, false, "", "Token exchange failed: "+html.EscapeString(err.Error()))
		return
	}
	if err := saveToken(tok); err != nil {
		writeOAuthPage(w, false, "", "Failed to save token: "+html.EscapeString(err.Error()))
		return
	}

	writeOAuthPage(w, true, tok.TeamName, "")
}

// Ingest handles POST /slack/ingest by ingesting a single Slack channel or message
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a Slack channel or message
// @Description  Fetches messages from a Slack channel by URI and returns a provenance-rich event.
// @Tags         slack
// @Accept       json
// @Produce      json
// @Param        body  body      request.SlackIngest   true  "Slack ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /slack/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.SlackIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	metadata := map[string]string{}
	if token := strings.TrimSpace(req.Token); token != "" {
		metadata["slack_token"] = token
	} else if tok, err := loadToken(); err == nil && tok.AccessToken != "" {
		metadata["slack_token"] = tok.AccessToken
	}
	// If neither is set, the connector falls back to SLACK_BOT_TOKEN env var.

	connector := contracts.MCPSourceConnector(slacksource.NewConnector())
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		connector = codexsource.NewConnector()
		metadata[codexsource.MetadataPlugin] = codexsource.PluginSlack
	}

	shared.WriteSourceIngest(w, r, connector, shared.SourceIngestInput{URI: req.URI, Metadata: metadata})
}

// IngestStream handles POST /slack/ingest/stream.
// It streams Codex CLI log lines as SSE "log" events while the process runs,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream Slack Codex ingest
// @Description  Streams Codex CLI progress via SSE, then emits a result event.
// @Tags         slack
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.SlackIngest  true  "Slack ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /slack/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginSlack,
		[]string{"messages"},
		func(dec *json.Decoder) (request.SlackIngest, error) {
			var req request.SlackIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.SlackIngest) string { return req.URI },
		func(req request.SlackIngest) string { return req.Provider },
		func(req request.SlackIngest) string { return req.Token },
	)
}

// ---- OAuth helpers ----------------------------------------------------------

func generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func purgeExpiredOAuthStates() {
	slackStateStore.Range(func(k, v any) bool {
		if s, ok := v.(slackStateEntry); ok && time.Since(s.createdAt) > 10*time.Minute {
			slackStateStore.Delete(k)
		}
		return true
	})
}

func redirectURI() string {
	if uri := strings.TrimSpace(os.Getenv("SLACK_REDIRECT_URI")); uri != "" {
		return uri
	}
	return slackDefaultRedirectURI
}

func exchangeCode(ctx context.Context, clientID, clientSecret, code string) (*slackStoredToken, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", redirectURI())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackOAuthTokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		Team        struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("slack oauth: %s", result.Error)
	}

	return &slackStoredToken{
		AccessToken: result.AccessToken,
		TeamID:      result.Team.ID,
		TeamName:    result.Team.Name,
		Scope:       result.Scope,
	}, nil
}

// ---- Token persistence ------------------------------------------------------

func tokenPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "context-os", "slack-token.json"), nil
}

func saveToken(tok *slackStoredToken) error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func loadToken() (*slackStoredToken, error) {
	path, err := tokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok slackStoredToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// ---- OAuth result page ------------------------------------------------------

func writeOAuthPage(w http.ResponseWriter, success bool, teamName, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !success {
		w.WriteHeader(http.StatusBadRequest)
	}

	var bodyHTML string
	var script string
	if success {
		bodyHTML = fmt.Sprintf(
			`<div class="box success"><span class="icon">&#10003;</span> Connected to <strong>%s</strong>.<br>You can close this window.</div>`,
			html.EscapeString(teamName),
		)
		script = `if(window.opener){window.opener.postMessage('slack_connected','*');window.close();}else{setTimeout(function(){window.location='/';},1500);}`
	} else {
		bodyHTML = fmt.Sprintf(
			`<div class="box error"><span class="icon">&#10007;</span> %s<br><a href="javascript:window.close()">Close</a></div>`,
			errMsg,
		)
	}

	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Slack OAuth</title>
<style>
  body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb;}
  .box{text-align:center;padding:2rem;border-radius:10px;max-width:360px;}
  .success{background:#f0fdf4;border:1px solid #bbf7d0;color:#166534;}
  .error{background:#fef2f2;border:1px solid #fecaca;color:#991b1b;}
  .icon{font-size:2rem;display:block;margin-bottom:.5rem;}
</style>
</head><body>%s<script>%s</script></body></html>`, bodyHTML, script)
}
