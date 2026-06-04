// Package jira provides HTTP handlers for the /jira/* routes.
package jira

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	codexsource "context-os/internal/source/codex"
	jirasource "context-os/internal/source/jira"
)

// Status handles GET /jira/status.
// It reports whether Jira base URL and authentication environment variables are configured.
//
// @Summary      Jira connection status
// @Description  Returns whether Jira base URL and authentication environment variables are configured.
// @Tags         jira
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /jira/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	baseURL := strings.TrimSpace(os.Getenv("JIRA_BASE_URL"))
	tokenConfigured := strings.TrimSpace(os.Getenv("JIRA_TOKEN")) != ""
	emailConfigured := strings.TrimSpace(os.Getenv("JIRA_EMAIL")) != ""

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":        baseURL != "",
		"base_url":         baseURL,
		"token_configured": tokenConfigured,
		"email_configured": emailConfigured,
	})
}

// Ingest handles POST /jira/ingest by ingesting a Jira issue or project
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a Jira issue or project
// @Description  Fetches Jira issue or project context by URI and returns a provenance-rich event.
// @Tags         jira
// @Accept       json
// @Produce      json
// @Param        body  body      request.JiraIngest  true  "Jira ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /jira/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.JiraIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	metadata := buildMetadata(req)
	connector := jirasource.NewConnector()
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		if strings.TrimSpace(req.URI) == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required for provider=codex")
			return
		}
		metadata = map[string]string{codexsource.MetadataPlugin: codexsource.PluginAtlassianRovo}
		connector = codexsource.NewConnector()
	}

	shared.WriteSourceIngest(w, r, connector, shared.SourceIngestInput{
		WorkspaceID: req.WorkspaceID,
		Connector:   "jira",
		URI:         req.URI,
		Content:     req.Content,
		Cursor:      req.Cursor,
		Metadata:    metadata,
	})
}

// IngestStream handles POST /jira/ingest/stream.
// It streams Codex CLI log lines from the Atlassian Rovo plugin as SSE "log" events,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream Jira Codex/Rovo ingest
// @Description  Streams Atlassian Rovo Codex plugin progress via SSE, then emits a result event.
// @Tags         jira
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.JiraIngest  true  "Jira ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /jira/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginAtlassianRovo,
		[]string{"issues"},
		func(dec *json.Decoder) (request.JiraIngest, error) {
			var req request.JiraIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.JiraIngest) string { return req.URI },
		func(req request.JiraIngest) string { return req.Provider },
		func(req request.JiraIngest) string { return req.Token },
		func(req request.JiraIngest) string { return req.WorkspaceID },
		func(request.JiraIngest) string { return "jira" },
	)
}

func buildMetadata(req request.JiraIngest) map[string]string {
	metadata := shared.CloneStringMap(req.Metadata)
	shared.SetMetadata(metadata, "jira_token", req.Token)
	shared.SetMetadata(metadata, "jira_email", req.Email)
	shared.SetMetadata(metadata, "jira_api_base_url", req.APIBaseURL)
	shared.SetMetadata(metadata, "jira_expand", req.Expand)
	return metadata
}
