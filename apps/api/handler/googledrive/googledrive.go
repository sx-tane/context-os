// Package googledrive provides HTTP handlers for the /googledrive/* routes.
package googledrive

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	codexsource "context-os/internal/source/codex"
	googledrivesource "context-os/internal/source/googledrive"
)

// Status handles GET /googledrive/status.
// It reports whether OAuth or service-account credentials are configured,
// and whether a default folder is available for ingest requests.
//
// @Summary      Google Drive connection status
// @Description  Returns whether OAuth credentials, service account credentials, or a direct access token are configured for Google Drive ingest.
// @Tags         googledrive
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /googledrive/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	oauthConfigured := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH")) != ""
	serviceAccountConfigured := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH")) != ""
	accessTokenConfigured := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_ACCESS_TOKEN")) != ""
	folderConfigured := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_FOLDER_ID")) != ""

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":                  oauthConfigured || serviceAccountConfigured || accessTokenConfigured,
		"oauth_configured":           oauthConfigured,
		"service_account_configured": serviceAccountConfigured,
		"access_token_configured":    accessTokenConfigured,
		"folder_configured":          folderConfigured,
	})
}

// Ingest handles POST /googledrive/ingest by listing supported files in a Google Drive folder,
// downloading their content, and returning one provenance-rich event per file.
// When provider is "codex" the request is routed through the Google Drive Codex plugin instead.
//
// @Summary      Ingest Google Drive documents
// @Description  Lists Google Docs, Sheets, and Slides in a configured folder and emits one raw ingest event per file. Set provider=codex to route through the Google Drive Codex plugin.
// @Tags         googledrive
// @Accept       json
// @Produce      json
// @Param        body  body      request.GoogleDriveIngest  true  "Google Drive ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /googledrive/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.GoogleDriveIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	uri := resolveURI(req)

	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		if uri == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required for provider=codex")
			return
		}
		shared.WriteSourceIngest(w, r, codexsource.NewConnector(), shared.SourceIngestInput{
			URI:      uri,
			Metadata: map[string]string{codexsource.MetadataPlugin: codexsource.PluginGoogleDrive},
		})
		return
	}

	metadata := shared.CloneStringMap(req.Metadata)
	shared.SetMetadata(metadata, googledrivesource.MetadataOAuthCredentialsPath, req.CredentialPath)
	shared.SetMetadata(metadata, googledrivesource.MetadataServiceAccountPath, req.ServiceAccountPath)
	shared.SetMetadata(metadata, googledrivesource.MetadataAccessToken, req.AccessToken)

	shared.WriteSourceIngest(w, r, googledrivesource.NewConnector(), shared.SourceIngestInput{
		URI:      uri,
		Cursor:   req.Cursor,
		Metadata: metadata,
	})
}

// IngestStream handles POST /googledrive/ingest/stream.
// It streams Google Drive Codex plugin progress as SSE "log" events,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream Google Drive Codex ingest
// @Description  Streams Google Drive Codex plugin progress via SSE, then emits a result event.
// @Tags         googledrive
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.GoogleDriveIngest  true  "Google Drive ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /googledrive/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	shared.StreamCodexIngest(
		w,
		r,
		codexsource.PluginGoogleDrive,
		[]string{"files"},
		func(dec *json.Decoder) (request.GoogleDriveIngest, error) {
			var req request.GoogleDriveIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.GoogleDriveIngest) string { return resolveURI(req) },
		func(req request.GoogleDriveIngest) string { return req.Provider },
		func(req request.GoogleDriveIngest) string { return "" },
	)
}

// resolveURI returns the drive folder URI from req.URI, req.FolderID, or the
// GOOGLE_DRIVE_FOLDER_ID environment variable, in that order of preference.
func resolveURI(req request.GoogleDriveIngest) string {
	if uri := strings.TrimSpace(req.URI); uri != "" {
		return uri
	}
	if folderID := strings.TrimSpace(req.FolderID); folderID != "" {
		return "https://drive.google.com/drive/folders/" + folderID
	}
	if folderID := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_FOLDER_ID")); folderID != "" {
		return "https://drive.google.com/drive/folders/" + folderID
	}
	return ""
}
