package googledrive

import (
	"context"
	"net/http"
	"time"

	"context-os/internal/source"
)

const (
	defaultDriveAPIBaseURL  = "https://www.googleapis.com/drive/v3"
	defaultSlidesAPIBaseURL = "https://slides.googleapis.com/v1"
	defaultGoogleTokenURL   = "https://oauth2.googleapis.com/token"
	defaultScope            = "https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/presentations.readonly"
	maxAttempts             = 4
	maxResponseBytes        = 20 << 20
	defaultFolderObjectType = "folder"
	defaultFileObjectType   = "file"

	metadataFolderID      = "googledrive_folder_id"
	metadataFileID        = "googledrive_file_id"
	metadataFileName      = "googledrive_file_name"
	metadataFileMimeType  = "googledrive_mime_type"
	metadataModifiedTime  = "googledrive_modified_time"
	metadataExportFormat  = "googledrive_export_format"
	metadataCredentialTyp = "googledrive_credential_type"
)

const (
	googleDocsMimeType         = "application/vnd.google-apps.document"
	googleSheetsMimeType       = "application/vnd.google-apps.spreadsheet"
	googleSlidesMimeType       = "application/vnd.google-apps.presentation"
	googleDocsExportMimeType   = "text/plain"
	googleSheetsExportMimeType = "text/csv"
)

const (
	googleDriveOAuthCredentialsEnv = "GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH"
	googleDriveServiceAccountEnv   = "GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH"
	googleDriveAccessTokenEnv      = "GOOGLE_DRIVE_ACCESS_TOKEN"
	googleDriveFolderIDEnv         = "GOOGLE_DRIVE_FOLDER_ID"
)

const (
	// MetadataOAuthCredentialsPath names the metadata key for an authorized-user OAuth credentials JSON path.
	MetadataOAuthCredentialsPath = "googledrive_oauth_credentials_path"
	// MetadataServiceAccountPath names the metadata key for a Google service account JSON path.
	MetadataServiceAccountPath = "googledrive_service_account_path"
	// MetadataAccessToken names the metadata key for a pre-issued Google OAuth access token override.
	MetadataAccessToken = "googledrive_access_token"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type connector struct {
	base             source.MCPConnector
	client           httpClient
	driveAPIBaseURL  string
	slidesAPIBaseURL string
	sleep            func(context.Context, time.Duration) error
}

type driveFile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	ModifiedTime string `json:"modifiedTime"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type authorizedUserCredentials struct {
	Type         string `json:"type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	TokenURI     string `json:"token_uri"`
}

type serviceAccountCredentials struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}
