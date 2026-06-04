// Package sharepoint provides HTTP handlers for the /sharepoint/* routes.
package sharepoint

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	codexsource "context-os/internal/source/codex"
	sharepointsource "context-os/internal/source/sharepoint"
)

// Status handles GET /sharepoint/status.
// It reports whether SharePoint / Microsoft Graph credentials are configured.
//
// @Summary      SharePoint connection status
// @Description  Returns whether a SharePoint access token or client credentials are configured for Microsoft Graph ingest.
// @Tags         sharepoint
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /sharepoint/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	accessTokenConfigured := strings.TrimSpace(os.Getenv("SHAREPOINT_ACCESS_TOKEN")) != ""
	tenantConfigured := strings.TrimSpace(os.Getenv("SHAREPOINT_TENANT_ID")) != ""
	clientIDConfigured := strings.TrimSpace(os.Getenv("SHAREPOINT_CLIENT_ID")) != ""
	clientSecretConfigured := strings.TrimSpace(os.Getenv("SHAREPOINT_CLIENT_SECRET")) != ""
	clientCredentialsConfigured := tenantConfigured && clientIDConfigured && clientSecretConfigured

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":                     accessTokenConfigured || clientCredentialsConfigured,
		"access_token_configured":       accessTokenConfigured,
		"client_credentials_configured": clientCredentialsConfigured,
		"tenant_configured":             tenantConfigured,
		"codex_plugin":                  "sharepoint@openai-curated",
	})
}

// Ingest handles POST /sharepoint/ingest by ingesting a SharePoint or OneDrive item
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a SharePoint / OneDrive item
// @Description  Fetches a SharePoint or OneDrive file by URI using Microsoft Graph and returns a provenance-rich event. Set provider=codex to route through the SharePoint Codex plugin.
// @Tags         sharepoint
// @Accept       json
// @Produce      json
// @Param        body  body      request.SharePointIngest  true  "SharePoint ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /sharepoint/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.SharePointIngest
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
			WorkspaceID: req.WorkspaceID,
			Connector:   "sharepoint",
			URI:         req.URI,
			Metadata:    map[string]string{codexsource.MetadataPlugin: codexsource.PluginSharePoint},
		})
		return
	}

	metadata := shared.CloneStringMap(req.Metadata)
	shared.SetMetadata(metadata, sharepointsource.MetadataAccessToken, req.Token)
	shared.SetMetadata(metadata, sharepointsource.MetadataTenantID, req.TenantID)
	shared.SetMetadata(metadata, sharepointsource.MetadataClientID, req.ClientID)
	shared.SetMetadata(metadata, sharepointsource.MetadataClientSecret, req.ClientSecret)

	shared.WriteSourceIngest(w, r, sharepointsource.NewConnector(), shared.SourceIngestInput{
		WorkspaceID: req.WorkspaceID,
		Connector:   "sharepoint",
		URI:         req.URI,
		Content:     req.Content,
		Metadata:    metadata,
	})
}

// IngestStream handles POST /sharepoint/ingest/stream.
// It streams SharePoint Codex plugin progress as SSE "log" events,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream SharePoint Codex ingest
// @Description  Streams SharePoint Codex plugin progress via SSE, then emits a result event.
// @Tags         sharepoint
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.SharePointIngest  true  "SharePoint ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /sharepoint/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginSharePoint,
		[]string{"sites", "drives"},
		func(dec *json.Decoder) (request.SharePointIngest, error) {
			var req request.SharePointIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.SharePointIngest) string { return req.URI },
		func(req request.SharePointIngest) string { return req.Provider },
		func(req request.SharePointIngest) string { return req.Token },
		func(req request.SharePointIngest) string { return req.WorkspaceID },
		func(request.SharePointIngest) string { return "sharepoint" },
	)
}
