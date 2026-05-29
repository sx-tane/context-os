// Package jira provides an MCP source connector for Jira issue tracker artifacts.
package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	metadataJiraIssueKey  = "jira_issue_key"
	metadataJiraProject   = "jira_project_key"
	metadataJiraHost      = "jira_host"
	metadataJiraAPIBase   = "jira_api_base_url"
	metadataJiraAPIStatus = "jira_api_status"
	metadataJiraUpdated   = "jira_updated"
	metadataJiraEmail     = "jira_email"
	metadataJiraToken     = "jira_token"
	metadataJiraExpand    = "jira_expand"

	defaultJiraExpand = "renderedFields,names,schema,changelog"
)

var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-[0-9]+$`)

type artifact struct {
	objectType string
	objectID   string
	sourceID   string
	issueKey   string
	projectKey string
	host       string
}

type connector struct {
	base   source.MCPConnector
	client httpClient
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// NewConnector returns a Jira source connector that ingests issue tracker events.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base: source.NewMCPConnector("jira", contracts.CapabilityIssues),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest enriches Jira issue or project metadata before emitting ingestion events.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, artifact{}, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	req.Metadata = cloneMetadata(req.Metadata)
	parsed, ok := parseArtifact(req.URI)
	if ok {
		enrichMetadata(parsed, req.Metadata)
	}

	if strings.TrimSpace(req.Content) == "" {
		if !ok {
			return nil, c.connectorError(req, parsed, contracts.ErrorKindInvalidRequest, false, errors.New("unsupported jira uri"))
		}
		if err := c.hydrateContent(ctx, &req, parsed); err != nil {
			return nil, err
		}
	}

	return c.base.Ingest(ctx, req)
}

func (c connector) hydrateContent(ctx context.Context, req *contracts.SourceRequest, parsed artifact) error {
	apiURL, err := buildAPIURL(parsed, req.Metadata)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}
	httpReq.Header.Set("Accept", "application/json")
	applyAuth(httpReq, req.Metadata)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return c.connectorError(*req, parsed, contracts.ErrorKindCanceled, true, err)
		}
		return c.connectorError(*req, parsed, contracts.ErrorKindTemporary, true, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindTemporary, true, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		kind, retryable := statusToError(resp.StatusCode)
		return c.connectorError(*req, parsed, kind, retryable, fmt.Errorf("jira api %s", strings.TrimSpace(string(body))))
	}

	req.Content = string(body)
	setIfMissing(req.Metadata, metadataJiraAPIStatus, strconv.Itoa(resp.StatusCode))
	if updated := updatedCursor(body); updated != "" {
		setIfMissing(req.Metadata, metadataJiraUpdated, updated)
		if req.Cursor == "" {
			req.Cursor = updated
		}
	}
	return nil
}

func buildAPIURL(parsed artifact, metadata map[string]string) (string, error) {
	base := strings.TrimSpace(metadata[metadataJiraAPIBase])
	if base == "" {
		base = parsed.host
	}
	if base == "" {
		base = strings.TrimSpace(os.Getenv("JIRA_BASE_URL"))
	}
	if base == "" {
		return "", errors.New("jira_api_base_url metadata or JIRA_BASE_URL is required")
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse jira api base url: %w", err)
	}

	path := "/rest/api/3/project/" + url.PathEscape(parsed.projectKey)
	if parsed.objectType == "issue" {
		path = "/rest/api/3/issue/" + url.PathEscape(parsed.issueKey)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + path
	if parsed.objectType == "issue" {
		expand := strings.TrimSpace(metadata[metadataJiraExpand])
		if expand == "" {
			expand = defaultJiraExpand
		}
		query := baseURL.Query()
		query.Set("expand", expand)
		baseURL.RawQuery = query.Encode()
	}
	return baseURL.String(), nil
}

func applyAuth(req *http.Request, metadata map[string]string) {
	token := strings.TrimSpace(metadata[metadataJiraToken])
	if token == "" {
		token = strings.TrimSpace(os.Getenv("JIRA_TOKEN"))
	}
	if token == "" {
		return
	}

	email := strings.TrimSpace(metadata[metadataJiraEmail])
	if email == "" {
		email = strings.TrimSpace(os.Getenv("JIRA_EMAIL"))
	}
	if email != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
		req.Header.Set("Authorization", "Basic "+encoded)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func statusToError(status int) (contracts.ErrorKind, bool) {
	if status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return contracts.ErrorKindTemporary, true
	}
	return contracts.ErrorKindPermanent, false
}

func updatedCursor(body []byte) string {
	var payload struct {
		Updated string `json:"updated"`
		Fields  struct {
			Updated string `json:"updated"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if updated := strings.TrimSpace(payload.Fields.Updated); updated != "" {
		return updated
	}
	return strings.TrimSpace(payload.Updated)
}

func parseArtifact(uri string) (artifact, bool) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return artifact{}, false
	}

	if strings.HasPrefix(trimmed, "jira://") {
		path := strings.Trim(strings.TrimPrefix(trimmed, "jira://"), "/")
		return parseJiraPath(path, "")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil || parsedURL.Host == "" {
		return artifact{}, false
	}

	host := parsedURL.Scheme + "://" + parsedURL.Host
	segments := splitPath(parsedURL.Path)
	for index, segment := range segments {
		if segment == "browse" && index+1 < len(segments) {
			return issueArtifact(segments[index+1], host)
		}
		if segment == "issue" && index+1 < len(segments) {
			return issueArtifact(segments[index+1], host)
		}
		if (segment == "projects" || segment == "project") && index+1 < len(segments) {
			return projectArtifact(segments[index+1], host)
		}
	}

	return artifact{}, false
}

func parseJiraPath(path, host string) (artifact, bool) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return artifact{}, false
	}

	if segments[0] == "issue" && len(segments) > 1 {
		return issueArtifact(segments[1], host)
	}
	if segments[0] == "project" && len(segments) > 1 {
		return projectArtifact(segments[1], host)
	}
	if issueKeyPattern.MatchString(strings.ToUpper(segments[0])) {
		return issueArtifact(segments[0], host)
	}
	return projectArtifact(segments[0], host)
}

func issueArtifact(key, host string) (artifact, bool) {
	issueKey := strings.ToUpper(strings.TrimSpace(key))
	if !issueKeyPattern.MatchString(issueKey) {
		return artifact{}, false
	}
	project := strings.SplitN(issueKey, "-", 2)[0]
	return artifact{
		objectType: "issue",
		objectID:   issueKey,
		sourceID:   "jira:issue:" + issueKey,
		issueKey:   issueKey,
		projectKey: project,
		host:       host,
	}, true
}

func projectArtifact(key, host string) (artifact, bool) {
	projectKey := strings.ToUpper(strings.TrimSpace(key))
	if projectKey == "" {
		return artifact{}, false
	}
	return artifact{
		objectType: "project",
		objectID:   projectKey,
		sourceID:   "jira:project:" + projectKey,
		projectKey: projectKey,
		host:       host,
	}, true
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

func enrichMetadata(parsed artifact, metadata map[string]string) {
	setIfMissing(metadata, contracts.MetadataObjectType, parsed.objectType)
	setIfMissing(metadata, contracts.MetadataObjectID, parsed.objectID)
	setIfMissing(metadata, events.MetadataSourceID, parsed.sourceID)
	setIfMissing(metadata, metadataJiraIssueKey, parsed.issueKey)
	setIfMissing(metadata, metadataJiraProject, parsed.projectKey)
	setIfMissing(metadata, metadataJiraHost, parsed.host)
}

func (c connector) connectorError(req contracts.SourceRequest, parsed artifact, kind contracts.ErrorKind, retryable bool, err error) error {
	objectType := parsed.objectType
	objectID := parsed.objectID
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

func setIfMissing(metadata map[string]string, key, value string) {
	if value == "" {
		return
	}
	if _, exists := metadata[key]; exists {
		return
	}
	metadata[key] = value
}
