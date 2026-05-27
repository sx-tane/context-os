package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	metadataGitHubOwner  = "github_owner"
	metadataGitHubRepo   = "github_repo"
	metadataGitHubNumber = "github_number"

	metadataGitHubToken        = "github_token"
	metadataGitHubAPIBaseURL   = "github_api_base_url"
	metadataGitHubAPIStatus    = "github_api_status"
	metadataGitHubETag         = "github_etag"
	metadataGitHubLastModified = "github_last_modified"
	defaultGitHubAPIBaseURL    = "https://api.github.com"
)

type artifact struct {
	objectType string
	objectID   string
	sourceID   string
	owner      string
	repo       string
	number     string
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type connector struct {
	base   source.MCPConnector
	client httpClient
}

// NewConnector returns a GitHub source connector that ingests repository, issue, and pull request artifacts.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base: source.NewMCPConnector("github", contracts.CapabilityRepository),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string {
	return c.base.Name()
}

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability {
	return c.base.Capabilities()
}

// Ingest enriches GitHub-specific provenance metadata before emitting ingestion events.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	req.Metadata = cloneMetadata(req.Metadata)
	enrichMetadata(req.URI, req.Metadata)

	if err := c.hydrateContent(ctx, &req); err != nil {
		return nil, err
	}

	return c.base.Ingest(ctx, req)
}

func (c connector) hydrateContent(ctx context.Context, req *contracts.SourceRequest) error {
	if strings.TrimSpace(req.Content) != "" {
		return nil
	}

	parsed, ok := parseArtifact(req.URI)
	if !ok {
		return nil
	}

	apiURL, err := apiURL(parsed, req.Metadata)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	if token := authToken(req.Metadata); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

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
		return c.connectorError(*req, parsed, kind, retryable, fmt.Errorf("github api %s", strings.TrimSpace(string(body))))
	}

	req.Content = string(body)
	setIfMissing(req.Metadata, metadataGitHubAPIStatus, strconv.Itoa(resp.StatusCode))
	setIfMissing(req.Metadata, metadataGitHubETag, strings.TrimSpace(resp.Header.Get("ETag")))
	setIfMissing(req.Metadata, metadataGitHubLastModified, strings.TrimSpace(resp.Header.Get("Last-Modified")))

	if req.Cursor == "" {
		var envelope struct {
			UpdatedAt string `json:"updated_at"`
		}
		if err := json.Unmarshal(body, &envelope); err == nil && strings.TrimSpace(envelope.UpdatedAt) != "" {
			req.Cursor = envelope.UpdatedAt
		}
	}

	return nil
}

func apiURL(parsed artifact, metadata map[string]string) (string, error) {
	baseURL := strings.TrimSpace(metadata[metadataGitHubAPIBaseURL])
	if baseURL == "" {
		baseURL = defaultGitHubAPIBaseURL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse github api base url: %w", err)
	}

	var path string
	switch parsed.objectType {
	case "issue":
		path = fmt.Sprintf("/repos/%s/%s/issues/%s", parsed.owner, parsed.repo, parsed.number)
	case "pull_request":
		path = fmt.Sprintf("/repos/%s/%s/pulls/%s", parsed.owner, parsed.repo, parsed.number)
	default:
		path = fmt.Sprintf("/repos/%s/%s", parsed.owner, parsed.repo)
	}

	base.Path = strings.TrimRight(base.Path, "/") + path
	return base.String(), nil
}

func statusToError(status int) (contracts.ErrorKind, bool) {
	if status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return contracts.ErrorKindTemporary, true
	}
	return contracts.ErrorKindPermanent, false
}

func authToken(metadata map[string]string) string {
	if token := strings.TrimSpace(metadata[metadataGitHubToken]); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
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

func enrichMetadata(uri string, metadata map[string]string) {
	parsed, ok := parseArtifact(uri)
	if !ok {
		return
	}

	setIfMissing(metadata, contracts.MetadataObjectType, parsed.objectType)
	setIfMissing(metadata, contracts.MetadataObjectID, parsed.objectID)
	setIfMissing(metadata, events.MetadataSourceID, parsed.sourceID)
	setIfMissing(metadata, metadataGitHubOwner, parsed.owner)
	setIfMissing(metadata, metadataGitHubRepo, parsed.repo)
	if parsed.number != "" {
		setIfMissing(metadata, metadataGitHubNumber, parsed.number)
	}
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

func parseArtifact(uri string) (artifact, bool) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return artifact{}, false
	}

	if strings.HasPrefix(trimmed, "repo://") {
		return parsePath(strings.TrimPrefix(trimmed, "repo://"))
	}
	if strings.HasPrefix(trimmed, "github://") {
		path := strings.TrimPrefix(trimmed, "github://")
		path = strings.TrimPrefix(path, "repos/")
		return parsePath(path)
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return artifact{}, false
	}

	switch parsedURL.Host {
	case "github.com":
		return parsePath(parsedURL.Path)
	case "api.github.com":
		path := strings.TrimPrefix(parsedURL.Path, "/repos/")
		return parsePath(path)
	default:
		return artifact{}, false
	}
}

func parsePath(path string) (artifact, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		return artifact{}, false
	}

	owner := segments[0]
	repo := segments[1]
	if owner == "" || repo == "" {
		return artifact{}, false
	}

	baseID := owner + "/" + repo
	parsed := artifact{
		objectType: "repository",
		objectID:   baseID,
		sourceID:   "github:repository:" + baseID,
		owner:      owner,
		repo:       repo,
	}

	if len(segments) < 4 || !isPositiveInt(segments[3]) {
		return parsed, true
	}

	number := segments[3]
	switch segments[2] {
	case "issues":
		parsed.objectType = "issue"
		parsed.objectID = baseID + "#" + number
		parsed.sourceID = "github:issue:" + parsed.objectID
		parsed.number = number
	case "pull", "pulls":
		parsed.objectType = "pull_request"
		parsed.objectID = baseID + "#" + number
		parsed.sourceID = "github:pull_request:" + parsed.objectID
		parsed.number = number
	}

	return parsed, true
}

func isPositiveInt(value string) bool {
	n, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return n > 0
}
