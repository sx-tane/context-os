// Package notion provides HTTP handlers for the /notion/* routes.
package notion

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	codexsource "context-os/internal/source/codex"
	notionsource "context-os/internal/source/notion"
)

// Status handles GET /notion/status.
// It reports whether a Notion integration token is configured.
//
// @Summary      Notion connection status
// @Description  Returns whether a Notion integration token environment variable is configured.
// @Tags         notion
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /notion/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	tokenConfigured := strings.TrimSpace(os.Getenv("NOTION_TOKEN")) != ""

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":        tokenConfigured,
		"token_configured": tokenConfigured,
		"codex_plugin":     "notion@openai-curated",
	})
}

// Ingest handles POST /notion/ingest by ingesting a Notion page or database
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a Notion page or database
// @Description  Fetches Notion page blocks or database entries by URI and returns a provenance-rich event. Set provider=codex to route through the Notion Codex plugin.
// @Tags         notion
// @Accept       json
// @Produce      json
// @Param        body  body      request.NotionIngest  true  "Notion ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /notion/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.NotionIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		if strings.TrimSpace(req.URI) == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required for provider=codex")
			return
		}
		shared.WriteSourceIngest(w, r, codexsource.NewConnector(), shared.SourceIngestInput{
			URI:      req.URI,
			Metadata: map[string]string{codexsource.MetadataPlugin: codexsource.PluginNotion},
		})
		return
	}

	metadata := shared.CloneStringMap(req.Metadata)
	shared.SetMetadata(metadata, notionsource.MetadataToken, req.Token)

	shared.WriteSourceIngest(w, r, notionsource.NewConnector(), shared.SourceIngestInput{
		URI:      req.URI,
		Content:  req.Content,
		Metadata: metadata,
	})
}

// IngestStream handles POST /notion/ingest/stream.
// It streams Notion Codex plugin progress as SSE "log" events,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream Notion Codex ingest
// @Description  Streams Notion Codex plugin progress via SSE, then emits a result event.
// @Tags         notion
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.NotionIngest  true  "Notion ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /notion/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginNotion,
		[]string{"pages", "databases"},
		func(dec *json.Decoder) (request.NotionIngest, error) {
			var req request.NotionIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.NotionIngest) string { return req.URI },
		func(req request.NotionIngest) string { return req.Provider },
		func(req request.NotionIngest) string { return req.Token },
	)
}
