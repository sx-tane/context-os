// Package sharepoint provides an MCP source connector for SharePoint and OneDrive via Microsoft Graph.
package sharepoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	defaultGraphAPIBase   = "https://graph.microsoft.com/v1.0"
	defaultTokenEndpoint  = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	graphScope            = "https://graph.microsoft.com/.default"
	maxResponseBytes      = 8 << 20
	maxContentBytes       = 4 << 20

	// MetadataAccessToken names the metadata key for a direct Graph API access token.
	MetadataAccessToken = "sharepoint_access_token"
	// MetadataTenantID names the metadata key for the Azure AD tenant ID.
	MetadataTenantID = "sharepoint_tenant_id"
	// MetadataClientID names the metadata key for the Azure AD application client ID.
	MetadataClientID = "sharepoint_client_id"
	// MetadataClientSecret names the metadata key for the Azure AD application client secret.
	MetadataClientSecret = "sharepoint_client_secret"

	metadataSiteID       = "sharepoint_site_id"
	metadataItemID       = "sharepoint_item_id"
	metadataItemName     = "sharepoint_item_name"
	metadataMimeType     = "sharepoint_mime_type"
	metadataETag         = "sharepoint_etag"
	metadataModifiedTime = "sharepoint_modified_time"
	metadataDriveID      = "sharepoint_drive_id"

	envAccessToken  = "SHAREPOINT_ACCESS_TOKEN"
	envTenantID     = "SHAREPOINT_TENANT_ID"
	envClientID     = "SHAREPOINT_CLIENT_ID"
	envClientSecret = "SHAREPOINT_CLIENT_SECRET"
)

// HTTPClient is the interface used for HTTP calls to the Microsoft Graph API.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type connector struct {
	base         source.MCPConnector
	client       HTTPClient
	graphAPIBase string
	tokenBase    string // base for token endpoint (allows test override)
}

// NewConnector returns a SharePoint source connector that ingests files via Microsoft Graph.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base:         source.NewMCPConnector("sharepoint", contracts.CapabilityFiles),
		client:       &http.Client{Timeout: 30 * time.Second},
		graphAPIBase: defaultGraphAPIBase,
		tokenBase:    "https://login.microsoftonline.com",
	}
}

// NewConnectorWithOptions returns a SharePoint connector with custom endpoints and client.
// Intended for tests that need to intercept HTTP calls.
func NewConnectorWithOptions(graphAPIBase, tokenBase string, client HTTPClient) contracts.MCPSourceConnector {
	return connector{
		base:         source.NewMCPConnector("sharepoint", contracts.CapabilityFiles),
		client:       client,
		graphAPIBase: graphAPIBase,
		tokenBase:    tokenBase,
	}
}

// Name returns the connector name.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

type parsedItem struct {
	siteID  string
	driveID string
	itemID  string
	rawURI  string
}

// parseArtifact extracts item IDs from a SharePoint URI.
// Supported formats:
//   - sharepoint://sites/{siteId}/items/{itemId}
//   - sharepoint://drives/{driveId}/items/{itemId}
//   - https://graph.microsoft.com/v1.0/sites/{siteId}/drive/items/{itemId}
func parseArtifact(uri string) (parsedItem, bool) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return parsedItem{}, false
	}

	if strings.HasPrefix(uri, "sharepoint://sites/") {
		rest := strings.TrimPrefix(uri, "sharepoint://sites/")
		parts := strings.SplitN(rest, "/items/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parsedItem{siteID: parts[0], itemID: parts[1], rawURI: uri}, true
		}
		return parsedItem{}, false
	}

	if strings.HasPrefix(uri, "sharepoint://drives/") {
		rest := strings.TrimPrefix(uri, "sharepoint://drives/")
		parts := strings.SplitN(rest, "/items/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parsedItem{driveID: parts[0], itemID: parts[1], rawURI: uri}, true
		}
		return parsedItem{}, false
	}

	if strings.Contains(uri, "graph.microsoft.com") {
		return parseGraphURL(uri)
	}

	return parsedItem{}, false
}

func parseGraphURL(uri string) (parsedItem, bool) {
	// e.g. https://graph.microsoft.com/v1.0/sites/{siteId}/drive/items/{itemId}
	for _, prefix := range []string{"https://graph.microsoft.com/v1.0/", "https://graph.microsoft.com/v1/"} {
		if strings.HasPrefix(uri, prefix) {
			path := strings.TrimPrefix(uri, prefix)
			if strings.HasPrefix(path, "sites/") {
				path = strings.TrimPrefix(path, "sites/")
				// path: {siteId}/drive/items/{itemId} or {siteId}/drives/{driveId}/items/{itemId}
				parts := strings.SplitN(path, "/", 2)
				siteID := parts[0]
				rest := parts[1]
				// find items/
				idx := strings.LastIndex(rest, "items/")
				if idx >= 0 {
					itemID := rest[idx+len("items/"):]
					item := parsedItem{siteID: siteID, itemID: itemID, rawURI: uri}
					if strings.Contains(rest, "drives/") {
						dparts := strings.SplitN(rest, "drives/", 2)
						driveAndRest := dparts[1]
						item.driveID = strings.SplitN(driveAndRest, "/", 2)[0]
					}
					return item, true
				}
			}
			if strings.HasPrefix(path, "drives/") {
				path = strings.TrimPrefix(path, "drives/")
				parts := strings.SplitN(path, "/items/", 2)
				if len(parts) == 2 {
					return parsedItem{driveID: parts[0], itemID: parts[1], rawURI: uri}, true
				}
			}
		}
	}
	return parsedItem{}, false
}

// Ingest fetches a SharePoint / OneDrive item and emits a provenance-rich event.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, connectorErr("sharepoint", req, contracts.ErrorKindCanceled, false, err)
	}

	req.Metadata = cloneMetadata(req.Metadata)
	item, ok := parseArtifact(req.URI)
	if !ok && strings.TrimSpace(req.Content) == "" {
		return nil, connectorErr("sharepoint", req, contracts.ErrorKindInvalidRequest, false,
			errors.New("unsupported sharepoint uri: use sharepoint://sites/{siteId}/items/{itemId} or sharepoint://drives/{driveId}/items/{itemId}"))
	}

	if ok {
		enrichItemMetadata(item, req.Metadata)
	}

	if strings.TrimSpace(req.Content) == "" {
		if err := c.hydrateContent(ctx, &req, item); err != nil {
			return nil, err
		}
	}

	return c.base.Ingest(ctx, req)
}

func enrichItemMetadata(item parsedItem, md map[string]string) {
	if item.siteID != "" {
		md[metadataSiteID] = item.siteID
		md[contracts.MetadataObjectType] = "site"
	}
	if item.driveID != "" {
		md[metadataDriveID] = item.driveID
		md[contracts.MetadataObjectType] = "drive"
	}
	if item.itemID != "" {
		md[metadataItemID] = item.itemID
		md[contracts.MetadataObjectID] = item.itemID
	}
}

func (c connector) hydrateContent(ctx context.Context, req *contracts.SourceRequest, item parsedItem) error {
	token, err := c.resolveToken(ctx, req.Metadata)
	if err != nil {
		return err
	}

	itemEndpoint := c.buildItemEndpoint(item)
	meta, err := c.fetchItemMeta(ctx, itemEndpoint, token)
	if err != nil {
		return err
	}

	req.Metadata[metadataItemName] = meta.Name
	req.Metadata[metadataMimeType] = meta.File.MimeType
	req.Metadata[metadataETag] = meta.ETag
	req.Metadata[metadataModifiedTime] = meta.LastModified
	if req.Cursor == "" {
		req.Cursor = meta.ETag
	}

	content, err := c.fetchItemContent(ctx, itemEndpoint, token, meta.File.MimeType)
	if err != nil {
		return err
	}
	req.Content = content
	return nil
}

type graphDriveItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ETag         string `json:"eTag"`
	LastModified string `json:"lastModifiedDateTime"`
	File         struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
	Size int64 `json:"size"`
}

func (c connector) buildItemEndpoint(item parsedItem) string {
	if item.siteID != "" && item.driveID != "" {
		return fmt.Sprintf("%s/sites/%s/drives/%s/items/%s", c.graphAPIBase, item.siteID, item.driveID, item.itemID)
	}
	if item.siteID != "" {
		return fmt.Sprintf("%s/sites/%s/drive/items/%s", c.graphAPIBase, item.siteID, item.itemID)
	}
	if item.driveID != "" {
		return fmt.Sprintf("%s/drives/%s/items/%s", c.graphAPIBase, item.driveID, item.itemID)
	}
	return ""
}

func (c connector) fetchItemMeta(ctx context.Context, endpoint, token string) (graphDriveItem, error) {
	resp, err := c.doGet(ctx, endpoint, token)
	if err != nil {
		return graphDriveItem{}, err
	}
	defer resp.Body.Close()

	var item graphDriveItem
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&item); err != nil {
		return graphDriveItem{}, fmt.Errorf("sharepoint: decode item metadata: %w", err)
	}
	return item, nil
}

func (c connector) fetchItemContent(ctx context.Context, itemEndpoint, token, mimeType string) (string, error) {
	contentEndpoint := itemEndpoint + "/content"
	resp, err := c.doGet(ctx, contentEndpoint, token)
	if err != nil {
		// For unsupported binary content, return empty — callers can inspect metadata
		return "", nil
	}
	defer resp.Body.Close()

	// For text-based files, return raw content
	if isTextMime(mimeType) {
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxContentBytes))
		if err != nil {
			return "", fmt.Errorf("sharepoint: read content: %w", err)
		}
		return string(data), nil
	}

	// PDF and other non-text: return metadata summary
	return fmt.Sprintf("[binary content: %s]", mimeType), nil
}

func isTextMime(mimeType string) bool {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return true
	case mimeType == "application/json":
		return true
	case mimeType == "application/xml":
		return true
	case mimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return false // Word — binary, would need extraction lib
	case mimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return false // Excel — binary
	default:
		return false
	}
}

func (c connector) doGet(ctx context.Context, endpoint, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("sharepoint: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sharepoint: request %s: %w", endpoint, err)
	}
	if err := checkStatus(resp, endpoint); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

func checkStatus(resp *http.Response, endpoint string) error {
	if resp.StatusCode < 400 {
		return nil
	}
	kind := contracts.ErrorKindPermanent
	retryable := false
	if resp.StatusCode >= 500 {
		kind = contracts.ErrorKindTemporary
		retryable = true
	}
	return &contracts.ConnectorError{
		Connector: "sharepoint",
		Kind:      kind,
		Retryable: retryable,
		Err:       fmt.Errorf("graph api %s returned %d", endpoint, resp.StatusCode),
	}
}

// resolveToken returns a valid Bearer token, either from metadata/env or via OAuth2 client credentials.
func (c connector) resolveToken(ctx context.Context, md map[string]string) (string, error) {
	// Direct access token
	if t := coalesce(md[MetadataAccessToken], os.Getenv(envAccessToken)); t != "" {
		return t, nil
	}
	// Client credentials
	tenantID := coalesce(md[MetadataTenantID], os.Getenv(envTenantID))
	clientID := coalesce(md[MetadataClientID], os.Getenv(envClientID))
	clientSecret := coalesce(md[MetadataClientSecret], os.Getenv(envClientSecret))

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return "", &contracts.ConnectorError{
			Connector: "sharepoint",
			Kind:      contracts.ErrorKindPermanent,
			Retryable: false,
			Err:       errors.New("no sharepoint credentials: set SHAREPOINT_ACCESS_TOKEN or SHAREPOINT_TENANT_ID + CLIENT_ID + CLIENT_SECRET"),
		}
	}

	return c.clientCredentialsGrant(ctx, tenantID, clientID, clientSecret)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func (c connector) clientCredentialsGrant(ctx context.Context, tenantID, clientID, clientSecret string) (string, error) {
	tokenURL := fmt.Sprintf("%s/%s/oauth2/v2.0/token", c.tokenBase, tenantID)

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", graphScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("sharepoint: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sharepoint: token request: %w", err)
	}
	defer resp.Body.Close()

	var tok tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&tok); err != nil {
		return "", fmt.Errorf("sharepoint: decode token response: %w", err)
	}
	if tok.Error != "" {
		return "", &contracts.ConnectorError{
			Connector: "sharepoint",
			Kind:      contracts.ErrorKindPermanent,
			Retryable: false,
			Err:       fmt.Errorf("sharepoint oauth2: %s: %s", tok.Error, tok.Description),
		}
	}
	if tok.AccessToken == "" {
		return "", &contracts.ConnectorError{
			Connector: "sharepoint",
			Kind:      contracts.ErrorKindPermanent,
			Retryable: false,
			Err:       errors.New("sharepoint oauth2: empty access token"),
		}
	}
	return tok.AccessToken, nil
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func cloneMetadata(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func connectorErr(name string, req contracts.SourceRequest, kind contracts.ErrorKind, retryable bool, err error) error {
	return &contracts.ConnectorError{
		Connector:  name,
		URI:        req.URI,
		ObjectType: req.Metadata[contracts.MetadataObjectType],
		ObjectID:   req.Metadata[contracts.MetadataObjectID],
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}

