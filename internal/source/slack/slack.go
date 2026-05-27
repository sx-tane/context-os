// Package slack provides a Slack MCP source connector that ingests channels and messages.
package slack

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
	metadataSlackChannelID  = "slack_channel_id"
	metadataSlackTS         = "slack_ts"
	metadataSlackAPIStatus  = "slack_api_status"
	metadataSlackToken      = "slack_token"
	metadataSlackAPIBaseURL = "slack_api_base_url"
	defaultSlackAPIBaseURL  = "https://slack.com/api"
)

type artifact struct {
	objectType string
	objectID   string
	sourceID   string
	channelID  string
	ts         string
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type connector struct {
	base   source.MCPConnector
	client httpClient
}

// NewConnector returns a Slack source connector that ingests channel and message artifacts.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base: source.NewMCPConnector("slack", contracts.CapabilityMessages),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest enriches Slack-specific provenance metadata before emitting ingestion events.
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

	apiURL, err := buildAPIURL(parsed, req.Metadata)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindInvalidRequest, false, err)
	}
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

	// Slack always returns HTTP 200 for API calls; network-level errors use other status codes.
	if resp.StatusCode != http.StatusOK {
		kind, retryable := httpStatusToError(resp.StatusCode)
		return c.connectorError(*req, parsed, kind, retryable, fmt.Errorf("slack api http %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	// Decode the Slack envelope to check the ok field before treating the body as content.
	var envelope slackEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return c.connectorError(*req, parsed, contracts.ErrorKindTemporary, true, fmt.Errorf("decode slack response: %w", err))
	}
	if !envelope.OK {
		kind, retryable := slackErrorToKind(envelope.Error)
		return c.connectorError(*req, parsed, kind, retryable, fmt.Errorf("slack api error: %s", envelope.Error))
	}

	req.Content = string(body)
	setIfMissing(req.Metadata, metadataSlackAPIStatus, "200")

	// Use the message ts as the replay cursor.
	if req.Cursor == "" && len(envelope.Messages) > 0 {
		if ts := strings.TrimSpace(envelope.Messages[0].TS); ts != "" {
			req.Cursor = ts
		}
	}

	return nil
}

// slackEnvelope captures the top-level fields common to all Slack API responses.
type slackEnvelope struct {
	OK       bool           `json:"ok"`
	Error    string         `json:"error"`
	Messages []slackMessage `json:"messages"`
	Channel  *slackChannel  `json:"channel"`
}

type slackMessage struct {
	TS   string `json:"ts"`
	Text string `json:"text"`
}

type slackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func buildAPIURL(parsed artifact, metadata map[string]string) (string, error) {
	base := strings.TrimSpace(metadata[metadataSlackAPIBaseURL])
	if base == "" {
		base = defaultSlackAPIBaseURL
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse slack api base url: %w", err)
	}

	var path string
	params := url.Values{}

	switch parsed.objectType {
	case "message":
		path = "/conversations.history"
		params.Set("channel", parsed.channelID)
		params.Set("latest", parsed.ts)
		params.Set("oldest", parsed.ts)
		params.Set("limit", "1")
		params.Set("inclusive", "true")
	default:
		path = "/conversations.info"
		params.Set("channel", parsed.channelID)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + path
	baseURL.RawQuery = params.Encode()
	return baseURL.String(), nil
}

func httpStatusToError(status int) (contracts.ErrorKind, bool) {
	if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return contracts.ErrorKindTemporary, true
	}
	return contracts.ErrorKindPermanent, false
}

func slackErrorToKind(slackErr string) (contracts.ErrorKind, bool) {
	switch slackErr {
	case "ratelimited":
		return contracts.ErrorKindTemporary, true
	case "not_authed", "invalid_auth", "account_inactive", "token_revoked",
		"channel_not_found", "message_not_found", "no_permission":
		return contracts.ErrorKindPermanent, false
	default:
		return contracts.ErrorKindTemporary, true
	}
}

func authToken(metadata map[string]string) string {
	if token := strings.TrimSpace(metadata[metadataSlackToken]); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("SLACK_BOT_TOKEN"))
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
	for k, v := range metadata {
		out[k] = v
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
	setIfMissing(metadata, metadataSlackChannelID, parsed.channelID)
	if parsed.ts != "" {
		setIfMissing(metadata, metadataSlackTS, parsed.ts)
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

// parseArtifact parses a Slack URI into an artifact.
// Supported forms:
//
//	slack://CHANNEL_ID           → channel artifact
//	slack://CHANNEL_ID/TS        → message artifact
func parseArtifact(uri string) (artifact, bool) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return artifact{}, false
	}

	if !strings.HasPrefix(trimmed, "slack://") {
		return artifact{}, false
	}

	path := strings.TrimPrefix(trimmed, "slack://")
	segments := strings.SplitN(strings.Trim(path, "/"), "/", 2)
	if len(segments) == 0 || segments[0] == "" {
		return artifact{}, false
	}

	channelID := segments[0]

	if len(segments) == 1 {
		return artifact{
			objectType: "channel",
			objectID:   channelID,
			sourceID:   "slack:channel:" + channelID,
			channelID:  channelID,
		}, true
	}

	ts := segments[1]
	if ts == "" {
		return artifact{
			objectType: "channel",
			objectID:   channelID,
			sourceID:   "slack:channel:" + channelID,
			channelID:  channelID,
		}, true
	}

	return artifact{
		objectType: "message",
		objectID:   channelID + ":" + ts,
		sourceID:   "slack:message:" + channelID + ":" + ts,
		channelID:  channelID,
		ts:         ts,
	}, true
}
