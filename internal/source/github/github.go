package github

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	metadataGitHubOwner  = "github_owner"
	metadataGitHubRepo   = "github_repo"
	metadataGitHubNumber = "github_number"
)

type artifact struct {
	objectType string
	objectID   string
	sourceID   string
	owner      string
	repo       string
	number     string
}

type connector struct {
	base source.MCPConnector
}

// NewConnector returns a GitHub source connector that ingests repository, issue, and pull request artifacts.
func NewConnector() contracts.MCPSourceConnector {
	return connector{base: source.NewMCPConnector("github", contracts.CapabilityRepository)}
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
	return c.base.Ingest(ctx, req)
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
