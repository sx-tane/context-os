// Package googledrive provides an MCP source connector for Google Drive artifacts.
package googledrive

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
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

// NewConnector returns a Google Drive source connector that ingests Docs, Sheets, and Slides.
func NewConnector() contracts.MCPSourceConnector {
	return newConnector(&http.Client{Timeout: 15 * time.Second}, defaultDriveAPIBaseURL, defaultSlidesAPIBaseURL, sleepContext)
}

func newConnector(client httpClient, driveAPIBaseURL, slidesAPIBaseURL string, sleep func(context.Context, time.Duration) error) connector {
	return connector{
		base:             source.NewMCPConnector("googledrive", contracts.CapabilityFiles),
		client:           client,
		driveAPIBaseURL:  strings.TrimRight(driveAPIBaseURL, "/"),
		slidesAPIBaseURL: strings.TrimRight(slidesAPIBaseURL, "/"),
		sleep:            sleep,
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest lists supported Google Drive files in a folder, downloads their text content, and emits one event per file.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, defaultFolderObjectType, "", contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	metadata := cloneMetadata(req.Metadata)
	if strings.TrimSpace(req.Content) != "" {
		req.Metadata = metadata
		return c.base.Ingest(ctx, req)
	}

	folderID, err := folderIDFromRequest(req.URI, metadata)
	if err != nil {
		return nil, c.connectorError(req, defaultFolderObjectType, "", contracts.ErrorKindInvalidRequest, false, err)
	}
	metadata[metadataFolderID] = folderID

	token, credentialType, err := c.accessToken(ctx, metadata)
	if err != nil {
		kind, retryable := classifyGoogleError(err)
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, kind, retryable, err)
	}
	metadata[metadataCredentialTyp] = credentialType

	files, err := c.listFiles(ctx, folderID, req.Cursor, token)
	if err != nil {
		kind, retryable := classifyGoogleError(err)
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, kind, retryable, err)
	}
	if len(files) == 0 {
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, contracts.ErrorKindPermanent, false, errors.New("google drive folder has no supported files"))
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].ID == files[j].ID {
			return files[i].ModifiedTime < files[j].ModifiedTime
		}
		return files[i].ID < files[j].ID
	})

	ingested := make([]events.Event, 0, len(files))
	for _, file := range files {
		event, eventErr := c.ingestFile(ctx, req, metadata, file, token)
		if eventErr != nil {
			kind, retryable := classifyGoogleError(eventErr)
			return nil, c.connectorError(req, defaultFileObjectType, file.ID, kind, retryable, eventErr)
		}
		ingested = append(ingested, event)
	}

	return ingested, nil
}

func (c connector) ingestFile(ctx context.Context, req contracts.SourceRequest, baseMetadata map[string]string, file driveFile, token string) (events.Event, error) {
	content, exportFormat, err := c.fileContent(ctx, file, token)
	if err != nil {
		return events.Event{}, err
	}

	metadata := cloneMetadata(baseMetadata)
	metadata[contracts.MetadataObjectType] = defaultFileObjectType
	metadata[contracts.MetadataObjectID] = file.ID
	metadata[events.MetadataSourceID] = "googledrive:file:" + file.ID
	metadata[events.MetadataEventID] = stableEventID(file.ID, file.ModifiedTime)
	metadata[metadataFileID] = file.ID
	metadata[metadataFileName] = file.Name
	metadata[metadataFileMimeType] = file.MimeType
	metadata[metadataModifiedTime] = file.ModifiedTime
	metadata[metadataExportFormat] = exportFormat
	metadata["url"] = driveFileURL(file.ID)

	baseReq := contracts.SourceRequest{
		URI:      driveFileURL(file.ID),
		Content:  content,
		Cursor:   file.ModifiedTime,
		Metadata: metadata,
	}
	created, err := c.base.Ingest(ctx, baseReq)
	if err != nil {
		return events.Event{}, err
	}
	if len(created) != 1 {
		return events.Event{}, errors.New("google drive connector expected one event per file")
	}
	return created[0], nil
}

func (c connector) fileContent(ctx context.Context, file driveFile, token string) (string, string, error) {
	switch file.MimeType {
	case googleDocsMimeType:
		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files/"+url.PathEscape(file.ID)+"/export?mimeType="+url.QueryEscape(googleDocsExportMimeType), token)
		if err != nil {
			return "", "", err
		}
		return string(body), googleDocsExportMimeType, nil
	case googleSheetsMimeType:
		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files/"+url.PathEscape(file.ID)+"/export?mimeType="+url.QueryEscape(googleSheetsExportMimeType), token)
		if err != nil {
			return "", "", err
		}
		formatted, formatErr := formatCSVAsTable(body)
		if formatErr != nil {
			return "", "", formatErr
		}
		return formatted, googleSheetsExportMimeType, nil
	case googleSlidesMimeType:
		body, _, err := c.get(ctx, c.slidesAPIBaseURL+"/presentations/"+url.PathEscape(file.ID), token)
		if err != nil {
			return "", "", err
		}
		formatted, formatErr := formatSlidesAsText(body)
		if formatErr != nil {
			return "", "", formatErr
		}
		return formatted, "slides:text", nil
	default:
		return "", "", fmt.Errorf("unsupported google drive mime type %q", file.MimeType)
	}
}

func (c connector) listFiles(ctx context.Context, folderID, cursor, token string) ([]driveFile, error) {
	query := []string{
		fmt.Sprintf("'%s' in parents", escapeDriveQuery(folderID)),
		"trashed = false",
		"(mimeType = '" + googleDocsMimeType + "' or mimeType = '" + googleSheetsMimeType + "' or mimeType = '" + googleSlidesMimeType + "')",
	}
	if trimmedCursor := strings.TrimSpace(cursor); trimmedCursor != "" {
		parsedCursor, err := time.Parse(time.RFC3339Nano, trimmedCursor)
		if err != nil {
			return nil, fmt.Errorf("parse cursor: %w", err)
		}
		query = append(query, fmt.Sprintf("modifiedTime > '%s'", parsedCursor.UTC().Format(time.RFC3339Nano)))
	}

	params := url.Values{}
	params.Set("fields", "nextPageToken,files(id,name,mimeType,modifiedTime)")
	params.Set("includeItemsFromAllDrives", "true")
	params.Set("supportsAllDrives", "true")
	params.Set("pageSize", "1000")
	params.Set("q", strings.Join(query, " and "))
	params.Set("orderBy", "modifiedTime,name")

	files := make([]driveFile, 0)
	nextPageToken := ""
	for {
		pageParams := url.Values{}
		for key, values := range params {
			copied := make([]string, len(values))
			copy(copied, values)
			pageParams[key] = copied
		}
		if nextPageToken != "" {
			pageParams.Set("pageToken", nextPageToken)
		}

		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files?"+pageParams.Encode(), token)
		if err != nil {
			return nil, err
		}

		var payload struct {
			NextPageToken string      `json:"nextPageToken"`
			Files         []driveFile `json:"files"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode drive file list: %w", err)
		}
		files = append(files, payload.Files...)
		if strings.TrimSpace(payload.NextPageToken) == "" {
			return files, nil
		}
		nextPageToken = payload.NextPageToken
	}
}

func (c connector) accessToken(ctx context.Context, metadata map[string]string) (string, string, error) {
	if token := strings.TrimSpace(metadata[MetadataAccessToken]); token != "" {
		return token, "access_token", nil
	}
	if token := strings.TrimSpace(os.Getenv(googleDriveAccessTokenEnv)); token != "" {
		return token, "access_token", nil
	}

	if serviceAccountPath := firstNonEmpty(metadata[MetadataServiceAccountPath], os.Getenv(googleDriveServiceAccountEnv)); serviceAccountPath != "" {
		token, err := c.serviceAccountToken(ctx, serviceAccountPath)
		return token, "service_account", err
	}

	if credentialsPath := firstNonEmpty(metadata[MetadataOAuthCredentialsPath], os.Getenv(googleDriveOAuthCredentialsEnv)); credentialsPath != "" {
		token, err := c.authorizedUserToken(ctx, credentialsPath)
		return token, "oauth", err
	}

	return "", "", errors.New("google drive credentials are not configured")
}

func (c connector) authorizedUserToken(ctx context.Context, credentialsPath string) (string, error) {
	body, err := os.ReadFile(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("read oauth credentials: %w", err)
	}

	var credentials authorizedUserCredentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return "", fmt.Errorf("decode oauth credentials: %w", err)
	}
	if credentials.Type != "authorized_user" {
		return "", errors.New("oauth credentials must be an authorized_user JSON file")
	}
	if credentials.ClientID == "" || credentials.ClientSecret == "" || credentials.RefreshToken == "" {
		return "", errors.New("oauth credentials require client_id, client_secret, and refresh_token")
	}

	form := url.Values{}
	form.Set("client_id", credentials.ClientID)
	form.Set("client_secret", credentials.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", credentials.RefreshToken)
	bodyBytes, _, err := c.postForm(ctx, firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL), form.Encode())
	if err != nil {
		return "", err
	}

	var token tokenResponse
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		return "", fmt.Errorf("decode oauth token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("oauth token response missing access_token")
	}
	return token.AccessToken, nil
}

func (c connector) serviceAccountToken(ctx context.Context, credentialsPath string) (string, error) {
	body, err := os.ReadFile(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("read service account credentials: %w", err)
	}

	var credentials serviceAccountCredentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return "", fmt.Errorf("decode service account credentials: %w", err)
	}
	if credentials.Type != "service_account" {
		return "", errors.New("service account credentials must be a service_account JSON file")
	}
	if credentials.ClientEmail == "" || credentials.PrivateKey == "" {
		return "", errors.New("service account credentials require client_email and private_key")
	}

	assertion, err := signedJWT(credentials)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	bodyBytes, _, err := c.postForm(ctx, firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL), form.Encode())
	if err != nil {
		return "", err
	}

	var token tokenResponse
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		return "", fmt.Errorf("decode service account token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("service account token response missing access_token")
	}
	return token.AccessToken, nil
}

func (c connector) get(ctx context.Context, endpoint, token string) ([]byte, http.Header, error) {
	return c.doRequest(ctx, http.MethodGet, endpoint, token, "", "")
}

func (c connector) postForm(ctx context.Context, endpoint, body string) ([]byte, http.Header, error) {
	return c.doRequest(ctx, http.MethodPost, endpoint, "", "application/x-www-form-urlencoded", body)
}

func (c connector) doRequest(ctx context.Context, method, endpoint, token, contentType, body string) ([]byte, http.Header, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		var requestBody io.Reader
		if body != "" {
			requestBody = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
		if err != nil {
			return nil, nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt == maxAttempts {
				return nil, nil, lastErr
			}
			if sleepErr := c.sleep(ctx, backoffDuration(attempt, nil)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		responseBody, readErr := readResponseBody(resp)
		closeErr := resp.Body.Close()
		if readErr != nil {
			if closeErr != nil {
				return nil, nil, errors.Join(readErr, closeErr)
			}
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, closeErr
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			lastErr = googleAPIError{status: resp.StatusCode, message: strings.TrimSpace(string(responseBody))}
			if attempt == maxAttempts {
				return nil, resp.Header.Clone(), lastErr
			}
			if sleepErr := c.sleep(ctx, backoffDuration(attempt, resp.Header)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return nil, resp.Header.Clone(), googleAPIError{status: resp.StatusCode, message: strings.TrimSpace(string(responseBody))}
		}
		return responseBody, resp.Header.Clone(), nil
	}

	if lastErr == nil {
		lastErr = errors.New("request failed without an explicit error")
	}
	return nil, nil, lastErr
}

type googleAPIError struct {
	status  int
	message string
}

func (e googleAPIError) Error() string {
	if e.message == "" {
		return fmt.Sprintf("google api status %d", e.status)
	}
	return fmt.Sprintf("google api status %d: %s", e.status, e.message)
}

func classifyGoogleError(err error) (contracts.ErrorKind, bool) {
	if err == nil {
		return contracts.ErrorKindPermanent, false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return contracts.ErrorKindCanceled, true
	}

	var apiErr googleAPIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.status == http.StatusTooManyRequests || apiErr.status >= http.StatusInternalServerError:
			return contracts.ErrorKindTemporary, true
		case apiErr.status == http.StatusForbidden:
			lower := strings.ToLower(apiErr.message)
			if strings.Contains(lower, "ratelimitexceeded") || strings.Contains(lower, "userratelimitexceeded") {
				return contracts.ErrorKindTemporary, true
			}
			return contracts.ErrorKindPermanent, false
		case apiErr.status == http.StatusUnauthorized || apiErr.status == http.StatusNotFound:
			return contracts.ErrorKindPermanent, false
		default:
			return contracts.ErrorKindInvalidRequest, false
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), "credentials") || strings.Contains(strings.ToLower(err.Error()), "token") {
		return contracts.ErrorKindPermanent, false
	}
	return contracts.ErrorKindTemporary, true
}

func folderIDFromRequest(uri string, metadata map[string]string) (string, error) {
	if folderID := strings.TrimSpace(metadata[metadataFolderID]); folderID != "" {
		return folderID, nil
	}
	if folderID := strings.TrimSpace(os.Getenv(googleDriveFolderIDEnv)); folderID != "" && strings.TrimSpace(uri) == "" {
		return folderID, nil
	}

	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return "", errors.New("google drive folder uri is required")
	}
	if !strings.Contains(trimmed, "://") && !strings.Contains(trimmed, "/") {
		return trimmed, nil
	}

	if strings.HasPrefix(trimmed, "googledrive://") || strings.HasPrefix(trimmed, "gdrive://") {
		trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "googledrive://"), "gdrive://")
		parts := splitPath(trimmed)
		if len(parts) >= 2 && parts[0] == "folder" {
			return parts[1], nil
		}
		if len(parts) == 1 {
			return parts[0], nil
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse google drive uri: %w", err)
	}
	if id := strings.TrimSpace(parsed.Query().Get("id")); id != "" {
		return id, nil
	}
	if !strings.EqualFold(parsed.Host, "drive.google.com") {
		return "", errors.New("google drive uri must point to drive.google.com or use googledrive://folder/<id>")
	}

	parts := splitPath(parsed.Path)
	for index, part := range parts {
		if part == "folders" && index+1 < len(parts) {
			return parts[index+1], nil
		}
	}
	return "", errors.New("google drive folder id not found in uri")
}

func signedJWT(credentials serviceAccountCredentials) (string, error) {
	block, _ := pem.Decode([]byte(credentials.PrivateKey))
	if block == nil {
		return "", errors.New("decode service account private key: missing pem block")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse service account private key: %w", err)
	}
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("service account private key must be RSA")
	}

	now := time.Now().UTC()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   credentials.ClientEmail,
		"scope": defaultScope,
		"aud":   firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt assertion: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func formatCSVAsTable(body []byte) (string, error) {
	reader := csv.NewReader(strings.NewReader(string(body)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse csv export: %w", err)
	}

	formatted := make([]string, 0, len(rows))
	for _, row := range rows {
		formatted = append(formatted, strings.Join(row, "\t"))
	}
	return strings.Join(formatted, "\n"), nil
}

func formatSlidesAsText(body []byte) (string, error) {
	var presentation struct {
		Slides []struct {
			PageElements []struct {
				Shape *struct {
					Text struct {
						TextElements []struct {
							TextRun *struct {
								Content string `json:"content"`
							} `json:"textRun"`
						} `json:"textElements"`
					} `json:"text"`
				} `json:"shape"`
			} `json:"pageElements"`
		} `json:"slides"`
	}
	if err := json.Unmarshal(body, &presentation); err != nil {
		return "", fmt.Errorf("decode slides presentation: %w", err)
	}

	sections := make([]string, 0, len(presentation.Slides))
	for index, slide := range presentation.Slides {
		lines := make([]string, 0)
		for _, element := range slide.PageElements {
			if element.Shape == nil {
				continue
			}
			for _, textElement := range element.Shape.Text.TextElements {
				if textElement.TextRun == nil {
					continue
				}
				content := strings.TrimSpace(textElement.TextRun.Content)
				if content == "" {
					continue
				}
				lines = append(lines, content)
			}
		}
		if len(lines) == 0 {
			continue
		}
		sections = append(sections, fmt.Sprintf("Slide %d\n%s", index+1, strings.Join(lines, "\n")))
	}
	return strings.Join(sections, "\n\n"), nil
}

func driveFileURL(fileID string) string {
	return "https://drive.google.com/file/d/" + url.PathEscape(fileID) + "/view"
}

func stableEventID(fileID, modifiedTime string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(fileID) + "\x00" + strings.TrimSpace(modifiedTime)))
	return "event:" + hex.EncodeToString(sum[:])
}

func backoffDuration(attempt int, headers http.Header) time.Duration {
	if headers != nil {
		if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" {
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				return seconds
			}
		}
	}
	return time.Duration(1<<(attempt-1)) * 200 * time.Millisecond
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	return body, nil
}

func (c connector) connectorError(req contracts.SourceRequest, objectType, objectID string, kind contracts.ErrorKind, retryable bool, err error) error {
	if objectType == "" {
		objectType = req.Metadata[contracts.MetadataObjectType]
	}
	if objectID == "" {
		objectID = req.Metadata[contracts.MetadataObjectID]
	}
	return &contracts.ConnectorError{
		Connector:  c.base.Name(),
		URI:        req.URI,
		ObjectType: objectType,
		ObjectID:   objectID,
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func escapeDriveQuery(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
