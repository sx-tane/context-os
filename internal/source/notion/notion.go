// Package notion provides an MCP source connector for Notion pages and databases.
package notion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	defaultNotionAPIBaseURL = "https://api.notion.com/v1"
	defaultNotionVersion    = "2022-06-28"
	maxResponseBytes        = 8 << 20
	defaultPageSize         = "100"

	metadataPageID     = "notion_page_id"
	metadataDatabaseID = "notion_database_id"
	metadataLastEdited = "notion_last_edited_time"
	metadataURL        = "notion_url"
	metadataTitle      = "notion_title"

	// MetadataToken names the metadata key for a per-request Notion integration token.
	MetadataToken = "notion_token"

	notionTokenEnv = "NOTION_TOKEN"

	objectTypePage     = "page"
	objectTypeDatabase = "database"
)

// HTTPClient is the interface used to make HTTP calls to the Notion API.
// Expose it so tests can inject a custom client without needing real network access.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type connector struct {
	base       source.MCPConnector
	client     HTTPClient
	apiBaseURL string
}

// NewConnector returns a Notion source connector that ingests pages and database entries.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base:       source.NewMCPConnector("notion", contracts.CapabilityDocs),
		client:     &http.Client{Timeout: 20 * time.Second},
		apiBaseURL: defaultNotionAPIBaseURL,
	}
}

// NewConnectorWithOptions returns a Notion connector with a custom API base URL and HTTP client.
// Intended for tests that need to intercept HTTP calls.
func NewConnectorWithOptions(apiBaseURL string, client HTTPClient) contracts.MCPSourceConnector {
	return connector{
		base:       source.NewMCPConnector("notion", contracts.CapabilityDocs),
		client:     client,
		apiBaseURL: apiBaseURL,
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

type parsedArtifact struct {
	objectType string
	objectID   string
	sourceID   string
}

// parseArtifact extracts a Notion page or database ID from a URI.
// Supported formats:
//   - notion://page/{id}
//   - notion://database/{id}
//   - https://www.notion.so/{title}-{id} (HTTPS page URL)
func parseArtifact(uri string) (parsedArtifact, bool) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return parsedArtifact{}, false
	}

	if strings.HasPrefix(uri, "notion://page/") {
		id := normalizeUUID(strings.TrimPrefix(uri, "notion://page/"))
		if id == "" {
			return parsedArtifact{}, false
		}
		return parsedArtifact{objectType: objectTypePage, objectID: id, sourceID: "notion:page:" + id}, true
	}

	if strings.HasPrefix(uri, "notion://database/") {
		id := normalizeUUID(strings.TrimPrefix(uri, "notion://database/"))
		if id == "" {
			return parsedArtifact{}, false
		}
		return parsedArtifact{objectType: objectTypeDatabase, objectID: id, sourceID: "notion:database:" + id}, true
	}

	if strings.Contains(uri, "notion.so/") {
		id := extractNotionURLID(uri)
		if id != "" {
			return parsedArtifact{objectType: objectTypePage, objectID: id, sourceID: "notion:page:" + id}, true
		}
	}

	return parsedArtifact{}, false
}

// normalizeUUID converts a 32-hex or already-hyphenated UUID string to standard form.
func normalizeUUID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return strings.ToLower(s)
	}
	bare := strings.ReplaceAll(s, "-", "")
	if len(bare) != 32 {
		return ""
	}
	for _, c := range bare {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return ""
		}
	}
	bare = strings.ToLower(bare)
	return fmt.Sprintf("%s-%s-%s-%s-%s", bare[0:8], bare[8:12], bare[12:16], bare[16:20], bare[20:32])
}

// extractNotionURLID extracts the trailing UUID from a Notion HTTPS URL.
func extractNotionURLID(uri string) string {
	// Strip query/fragment
	if i := strings.IndexAny(uri, "?#"); i >= 0 {
		uri = uri[:i]
	}
	parts := strings.Split(uri, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		seg := parts[i]
		// Segment may be "{title}-{id}" — last 32 hex chars after stripping dashes from title
		bare := strings.ReplaceAll(seg, "-", "")
		if len(bare) >= 32 {
			candidate := bare[len(bare)-32:]
			if id := normalizeUUID(candidate); id != "" {
				return id
			}
		}
	}
	return ""
}

func enrichMetadata(artifact parsedArtifact, md map[string]string) {
	md[contracts.MetadataObjectType] = artifact.objectType
	md[contracts.MetadataObjectID] = artifact.objectID
	switch artifact.objectType {
	case objectTypePage:
		md[metadataPageID] = artifact.objectID
	case objectTypeDatabase:
		md[metadataDatabaseID] = artifact.objectID
	}
}

// Ingest fetches Notion page blocks or database entries and emits a provenance-rich event.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, parsedArtifact{}, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	req.Metadata = cloneMetadata(req.Metadata)
	artifact, ok := parseArtifact(req.URI)
	if ok {
		enrichMetadata(artifact, req.Metadata)
	}

	if strings.TrimSpace(req.Content) == "" {
		if !ok {
			return nil, c.connectorError(req, parsedArtifact{}, contracts.ErrorKindInvalidRequest, false,
				errors.New("unsupported notion uri: use notion://page/{id} or notion://database/{id}"))
		}
		if err := c.hydrateContent(ctx, &req, artifact); err != nil {
			return nil, err
		}
	}

	return c.base.Ingest(ctx, req)
}

func (c connector) hydrateContent(ctx context.Context, req *contracts.SourceRequest, artifact parsedArtifact) error {
	token := resolveToken(req.Metadata)
	if token == "" {
		return c.connectorError(*req, artifact, contracts.ErrorKindPermanent, false,
			errors.New("no notion token: set NOTION_TOKEN env or pass notion_token in metadata"))
	}
	switch artifact.objectType {
	case objectTypePage:
		return c.hydratePage(ctx, req, artifact, token)
	case objectTypeDatabase:
		return c.hydrateDatabase(ctx, req, artifact, token)
	default:
		return c.connectorError(*req, artifact, contracts.ErrorKindInvalidRequest, false,
			fmt.Errorf("unsupported notion object type %q", artifact.objectType))
	}
}

func resolveToken(md map[string]string) string {
	if t := strings.TrimSpace(md[MetadataToken]); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv(notionTokenEnv))
}

type notionPage struct {
	ID             string         `json:"id"`
	LastEditedTime string         `json:"last_edited_time"`
	URL            string         `json:"url"`
	Properties     map[string]any `json:"properties"`
}

type notionBlock struct {
	Type           string         `json:"type"`
	Paragraph      *richTextBlock `json:"paragraph"`
	Heading1       *richTextBlock `json:"heading_1"`
	Heading2       *richTextBlock `json:"heading_2"`
	Heading3       *richTextBlock `json:"heading_3"`
	BulletItem     *richTextBlock `json:"bulleted_list_item"`
	NumberItem     *richTextBlock `json:"numbered_list_item"`
	ToDo           *richTextBlock `json:"to_do"`
	Toggle         *richTextBlock `json:"toggle"`
	Quote          *richTextBlock `json:"quote"`
	Code           *richTextBlock `json:"code"`
}

type richTextBlock struct {
	RichText []richText `json:"rich_text"`
}

type richText struct {
	PlainText string `json:"plain_text"`
}

type blocksResponse struct {
	Results    []notionBlock `json:"results"`
	HasMore    bool          `json:"has_more"`
	NextCursor string        `json:"next_cursor"`
}

type databaseQueryResult struct {
	Results []struct {
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties"`
	} `json:"results"`
}

func (c connector) hydratePage(ctx context.Context, req *contracts.SourceRequest, artifact parsedArtifact, token string) error {
	page, err := c.fetchPage(ctx, artifact.objectID, token)
	if err != nil {
		return err
	}
	req.Metadata[metadataLastEdited] = page.LastEditedTime
	req.Metadata[metadataURL] = page.URL
	if t := extractPageTitle(page); t != "" {
		req.Metadata[metadataTitle] = t
	}
	if req.Cursor == "" {
		req.Cursor = page.LastEditedTime
	}

	content, err := c.fetchBlocks(ctx, artifact.objectID, token)
	if err != nil {
		return err
	}
	req.Content = content
	return nil
}

func (c connector) hydrateDatabase(ctx context.Context, req *contracts.SourceRequest, artifact parsedArtifact, token string) error {
	entries, err := c.queryDatabase(ctx, artifact.objectID, token)
	if err != nil {
		return err
	}
	req.Content = entries
	return nil
}

func (c connector) fetchPage(ctx context.Context, pageID, token string) (notionPage, error) {
	endpoint := fmt.Sprintf("%s/pages/%s", c.apiBaseURL, pageID)
	resp, err := c.doGet(ctx, endpoint, token)
	if err != nil {
		return notionPage{}, err
	}
	defer resp.Body.Close()

	var page notionPage
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&page); err != nil {
		return notionPage{}, fmt.Errorf("notion: decode page %s: %w", pageID, err)
	}
	return page, nil
}

func (c connector) fetchBlocks(ctx context.Context, blockID, token string) (string, error) {
	var lines []string
	cursor := ""
	for {
		endpoint := fmt.Sprintf("%s/blocks/%s/children?page_size=%s", c.apiBaseURL, blockID, defaultPageSize)
		if cursor != "" {
			endpoint += "&start_cursor=" + cursor
		}
		resp, err := c.doGet(ctx, endpoint, token)
		if err != nil {
			return "", err
		}
		var result blocksResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("notion: decode blocks for %s: %w", blockID, err)
		}
		resp.Body.Close()

		for _, block := range result.Results {
			if text := extractBlockText(block); text != "" {
				lines = append(lines, text)
			}
		}
		if !result.HasMore {
			break
		}
		cursor = result.NextCursor
	}
	return strings.Join(lines, "\n"), nil
}

func (c connector) queryDatabase(ctx context.Context, databaseID, token string) (string, error) {
	endpoint := fmt.Sprintf("%s/databases/%s/query", c.apiBaseURL, databaseID)
	body := strings.NewReader(`{"page_size":100}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("notion: build database query: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", defaultNotionVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("notion: database query %s: %w", databaseID, err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, "notion", objectTypeDatabase, databaseID); err != nil {
		return "", err
	}

	var result databaseQueryResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return "", fmt.Errorf("notion: decode database result: %w", err)
	}

	var lines []string
	for _, entry := range result.Results {
		if line := extractPropertiesText(entry.Properties); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (c connector) doGet(ctx context.Context, endpoint, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("notion: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", defaultNotionVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion: request %s: %w", endpoint, err)
	}
	if err := checkStatus(resp, "notion", "", ""); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

func checkStatus(resp *http.Response, connector, objectType, objectID string) error {
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
		Connector:  connector,
		ObjectType: objectType,
		ObjectID:   objectID,
		Kind:       kind,
		Retryable:  retryable,
		Err:        fmt.Errorf("%s API returned %d", connector, resp.StatusCode),
	}
}

func extractBlockText(block notionBlock) string {
	var rc *richTextBlock
	switch block.Type {
	case "paragraph":
		rc = block.Paragraph
	case "heading_1":
		rc = block.Heading1
	case "heading_2":
		rc = block.Heading2
	case "heading_3":
		rc = block.Heading3
	case "bulleted_list_item":
		rc = block.BulletItem
	case "numbered_list_item":
		rc = block.NumberItem
	case "to_do":
		rc = block.ToDo
	case "toggle":
		rc = block.Toggle
	case "quote":
		rc = block.Quote
	case "code":
		rc = block.Code
	}
	if rc == nil {
		return ""
	}
	var parts []string
	for _, rt := range rc.RichText {
		if rt.PlainText != "" {
			parts = append(parts, rt.PlainText)
		}
	}
	return strings.Join(parts, "")
}

func extractPageTitle(page notionPage) string {
	for _, prop := range page.Properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		if propMap["type"] != "title" {
			continue
		}
		titleArr, ok := propMap["title"].([]any)
		if !ok {
			continue
		}
		var parts []string
		for _, item := range titleArr {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := itemMap["plain_text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

func extractPropertiesText(props map[string]any) string {
	var parts []string
	for key, val := range props {
		propMap, ok := val.(map[string]any)
		if !ok {
			continue
		}
		propType, _ := propMap["type"].(string)
		if propType != "title" && propType != "rich_text" {
			continue
		}
		arr, ok := propMap[propType].([]any)
		if !ok {
			continue
		}
		var texts []string
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["plain_text"].(string); ok && t != "" {
				texts = append(texts, t)
			}
		}
		if len(texts) > 0 {
			parts = append(parts, key+": "+strings.Join(texts, ""))
		}
	}
	return strings.Join(parts, " | ")
}

func cloneMetadata(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func (c connector) connectorError(req contracts.SourceRequest, artifact parsedArtifact, kind contracts.ErrorKind, retryable bool, err error) error {
	objectType := req.Metadata[contracts.MetadataObjectType]
	objectID := req.Metadata[contracts.MetadataObjectID]
	if objectType == "" {
		objectType = artifact.objectType
	}
	if objectID == "" {
		objectID = artifact.objectID
	}
	return &contracts.ConnectorError{
		Connector:  "notion",
		URI:        req.URI,
		ObjectType: objectType,
		ObjectID:   objectID,
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}

