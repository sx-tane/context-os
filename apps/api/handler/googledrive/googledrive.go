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
//
// @Summary      Ingest Google Drive documents
// @Description  Lists Google Docs, Sheets, and Slides in a configured folder and emits one raw ingest event per file.
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
	shared.RunSourceIngest(w, r, googledrivesource.NewConnector(), func(dec *json.Decoder) (shared.SourceIngestInput, error) {
		var req request.GoogleDriveIngest
		if err := dec.Decode(&req); err != nil {
			return shared.SourceIngestInput{}, err
		}

		metadata := shared.CloneStringMap(req.Metadata)
		shared.SetMetadata(metadata, googledrivesource.MetadataOAuthCredentialsPath, req.CredentialPath)
		shared.SetMetadata(metadata, googledrivesource.MetadataServiceAccountPath, req.ServiceAccountPath)
		shared.SetMetadata(metadata, googledrivesource.MetadataAccessToken, req.AccessToken)

		uri := strings.TrimSpace(req.URI)
		if uri == "" {
			if folderID := strings.TrimSpace(req.FolderID); folderID != "" {
				uri = "https://drive.google.com/drive/folders/" + folderID
			} else if folderID := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_FOLDER_ID")); folderID != "" {
				uri = "https://drive.google.com/drive/folders/" + folderID
			}
		}

		return shared.SourceIngestInput{
			URI:      uri,
			Cursor:   req.Cursor,
			Metadata: metadata,
		}, nil
	})
}
