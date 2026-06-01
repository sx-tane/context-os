// Package chat answers local user questions from workspace-scoped ContextOS repositories.
package chat

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"context-os/domain/repository"
)

const (
	intentArtifacts   = "artifacts"
	intentFindings    = "findings"
	intentStatus      = "status"
	intentUnsupported = "unsupported"
	defaultLimit      = 20
	maxLimit          = 100
)

var (
	// ErrWorkspaceRequired is returned when a chat query omits workspace scope.
	ErrWorkspaceRequired = errors.New("workspace is required")
	// ErrWorkspaceNotFound is returned when a chat query references an unknown workspace.
	ErrWorkspaceNotFound = errors.New("workspace not found")
	// ErrMessageRequired is returned when a chat query omits the user message.
	ErrMessageRequired = errors.New("message is required")

	sourcePattern = regexp.MustCompile(`(?i)(#[A-Za-z0-9_.-]+|[a-z]+://[^\s,]+|https?://[^\s,]+|[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`)
)

// Query is one local chat request.
type Query struct {
	WorkspaceID   string
	WorkspacePath string
	Message       string
	Connector     string
	SourceURI     string
	Timezone      string
	LocalDate     string
	Limit         int
}

// Result is one deterministic local chat answer.
type Result struct {
	Intent        string
	WorkspaceID   string
	WorkspacePath string
	Connector     string
	SourceURI     string
	Answer        string
	Summary       string
	Since         *time.Time
	Until         *time.Time
	Artifacts     []repository.IngestEvent
	Syncs         []repository.ConnectorSync
}

// Service routes local chat queries to workspace repositories.
type Service struct {
	workspaces repository.WorkspaceRepository
	events     repository.EventRepository
	syncs      repository.SyncRepository
}

// NewService returns a Service backed by the provided repositories.
func NewService(workspaces repository.WorkspaceRepository, events repository.EventRepository, syncs repository.SyncRepository) *Service {
	return &Service{workspaces: workspaces, events: events, syncs: syncs}
}

// Query answers a user message using local workspace data only.
func (s *Service) Query(ctx context.Context, query Query) (Result, error) {
	message := strings.TrimSpace(query.Message)
	if message == "" {
		return Result{}, ErrMessageRequired
	}

	workspace, err := s.resolveWorkspace(ctx, query)
	if err != nil {
		return Result{}, err
	}

	syncs, err := s.listSyncs(ctx, workspace.ID)
	if err != nil {
		return Result{}, err
	}

	connector := normalizeConnector(firstNonEmpty(query.Connector, inferConnector(message)))
	sourceURI := firstNonEmpty(strings.TrimSpace(query.SourceURI), inferSourceURI(message))
	since, until := inferTimeRange(query, message)
	intent := classifyIntent(message, connector, sourceURI)

	result := Result{
		Intent:        intent,
		WorkspaceID:   workspace.ID,
		WorkspacePath: workspace.Path,
		Connector:     connector,
		SourceURI:     sourceURI,
		Since:         since,
		Until:         until,
		Syncs:         syncs,
	}

	switch intent {
	case intentArtifacts:
		return s.answerArtifacts(ctx, workspace.ID, query, result, message)
	case intentStatus:
		result.Answer = buildStatusAnswer(syncs)
		result.Summary = result.Answer
		return result, nil
	case intentFindings:
		result.Answer = "Findings are a local analysis view. Use the findings action to run or refresh mismatch detection, then inspect the evidence and graph in the truth panel."
		result.Summary = result.Answer
		return result, nil
	default:
		result.Answer = buildUnsupportedAnswer(syncs)
		result.Summary = result.Answer
		return result, nil
	}
}

func (s *Service) answerArtifacts(ctx context.Context, workspaceID string, query Query, result Result, message string) (Result, error) {
	limit := clampLimit(query.Limit)
	eventQuery := repository.EventQuery{
		Connector: result.Connector,
		SourceURI: result.SourceURI,
		Text:      inferSearchText(message),
		Since:     result.Since,
		Until:     result.Until,
		Limit:     limit,
	}

	artifacts, err := s.events.Query(ctx, workspaceID, eventQuery)
	if err != nil {
		return Result{}, fmt.Errorf("chat: query artifacts: %w", err)
	}
	result.Artifacts = artifacts
	result.Answer, result.Summary = buildArtifactAnswer(result, limit)
	return result, nil
}

func (s *Service) resolveWorkspace(ctx context.Context, query Query) (repository.Workspace, error) {
	ref := strings.TrimSpace(firstNonEmpty(query.WorkspacePath, query.WorkspaceID))
	if ref == "" {
		return repository.Workspace{}, ErrWorkspaceRequired
	}

	workspace, err := s.workspaces.GetByPath(ctx, ref)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("chat: get workspace by path: %w", err)
	}
	if workspace != nil {
		return *workspace, nil
	}

	workspaces, err := s.workspaces.List(ctx)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("chat: list workspaces: %w", err)
	}
	for _, workspace := range workspaces {
		if workspace.ID == ref || workspace.Path == ref {
			return workspace, nil
		}
	}
	return repository.Workspace{}, ErrWorkspaceNotFound
}

func (s *Service) listSyncs(ctx context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	if s.syncs == nil {
		return nil, nil
	}
	syncs, err := s.syncs.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("chat: list syncs: %w", err)
	}
	return syncs, nil
}

func classifyIntent(message, connector, sourceURI string) string {
	lower := strings.ToLower(message)
	if hasAny(lower, "status", "ready", "configured", "synced", "sync state") {
		return intentStatus
	}
	if connector != "" || sourceURI != "" || hasAny(lower, "message", "messages", "document", "documents", "doc", "docs", "issue", "issues", "ticket", "tickets", "pull request", "pr ", "channel", "source", "artifact", "artifacts", "today", "yesterday", "recent") {
		return intentArtifacts
	}
	if hasAny(lower, "finding", "findings", "mismatch", "mismatches", "risk", "risks", "analyze", "analyse", "analysis") {
		return intentFindings
	}
	return intentUnsupported
}

func inferConnector(message string) string {
	lower := strings.ToLower(message)
	checks := []struct {
		name  string
		terms []string
	}{
		{name: "googledrive", terms: []string{"google drive", "googledrive", "gdrive"}},
		{name: "sharepoint", terms: []string{"sharepoint", "one drive", "onedrive"}},
		{name: "github", terms: []string{"github", "pull request", "pr ", "repo", "commit"}},
		{name: "slack", terms: []string{"slack", "channel"}},
		{name: "jira", terms: []string{"jira", "ticket", "issue", "sprint"}},
		{name: "notion", terms: []string{"notion"}},
		{name: "filesystem", terms: []string{"filesystem", "file system", "local file", "docs/"}},
	}
	for _, check := range checks {
		for _, term := range check.terms {
			if strings.Contains(lower, term) {
				return check.name
			}
		}
	}
	return ""
}

func normalizeConnector(connector string) string {
	connector = strings.ToLower(strings.TrimSpace(connector))
	connector = strings.ReplaceAll(connector, "google-drive", "googledrive")
	connector = strings.ReplaceAll(connector, "google_drive", "googledrive")
	return connector
}

func inferSourceURI(message string) string {
	match := sourcePattern.FindString(message)
	return strings.Trim(match, `.,;:"'()[]{} `)
}

func inferSearchText(message string) string {
	lower := strings.ToLower(message)
	for _, marker := range []string{"containing ", "contains ", "mentioning ", "mentions ", "about "} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		text := strings.TrimSpace(message[idx+len(marker):])
		text = strings.Trim(text, `.,;:"'()[]{} `)
		if len(text) > 2 && !hasAny(strings.ToLower(text), " it", "this", "that") {
			return text
		}
	}
	return ""
}

func inferTimeRange(query Query, message string) (*time.Time, *time.Time) {
	lower := strings.ToLower(message)
	location := loadLocation(query.Timezone)
	base := localDate(query.LocalDate, location)

	if strings.Contains(lower, "yesterday") {
		start := time.Date(base.Year(), base.Month(), base.Day()-1, 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	if strings.Contains(lower, "today") {
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	if strings.Contains(lower, "this week") || strings.Contains(lower, "week") {
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -6)
		end := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	return nil, nil
}

func loadLocation(name string) *time.Location {
	if strings.TrimSpace(name) == "" {
		return time.Local
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return location
}

func localDate(raw string, location *time.Location) time.Time {
	if raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, location)
		if err == nil {
			return parsed
		}
	}
	return time.Now().In(location)
}

func utcPtr(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}

func buildArtifactAnswer(result Result, limit int) (string, string) {
	description := describeScope(result)
	if len(result.Artifacts) == 0 {
		answer := "No local " + description + " artifacts were found. Connect or sync that source, then ask again."
		return answer, answer
	}

	latest := result.Artifacts[0].Title
	if latest == "" {
		latest = previewText(result.Artifacts[0].Body, 90)
	}
	answer := fmt.Sprintf("Found %d local %s artifacts. Latest: %s", len(result.Artifacts), description, latest)
	if len(result.Artifacts) == limit {
		answer += fmt.Sprintf(" Showing the latest %d results.", limit)
	}
	return answer, summarizeArtifacts(result.Artifacts)
}

func describeScope(result Result) string {
	parts := []string{}
	if result.Connector != "" {
		parts = append(parts, result.Connector)
	} else {
		parts = append(parts, "source")
	}
	if result.SourceURI != "" {
		parts = append(parts, "from "+result.SourceURI)
	}
	if result.Since != nil && result.Until != nil {
		parts = append(parts, "for "+result.Since.Format("2006-01-02"))
	}
	return strings.Join(parts, " ")
}

func summarizeArtifacts(artifacts []repository.IngestEvent) string {
	if len(artifacts) == 0 {
		return ""
	}
	limit := len(artifacts)
	if limit > 5 {
		limit = 5
	}
	items := make([]string, 0, limit)
	for _, artifact := range artifacts[:limit] {
		title := strings.TrimSpace(artifact.Title)
		if title == "" {
			title = strings.TrimSpace(artifact.Body)
		}
		items = append(items, previewText(title, 90))
	}
	return "Latest local artifacts: " + strings.Join(items, " | ")
}

func previewText(text string, limit int) string {
	preview := strings.Join(strings.Fields(text), " ")
	if len(preview) <= limit {
		return preview
	}
	if limit <= 3 {
		return preview[:limit]
	}
	return preview[:limit-3] + "..."
}

func buildStatusAnswer(syncs []repository.ConnectorSync) string {
	if len(syncs) == 0 {
		return "No local sources are configured for this workspace yet. Add a source to build the local truth store."
	}
	counts := map[string]int{}
	for _, sync := range syncs {
		counts[sync.Status]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	return "Local source status: " + strings.Join(parts, ", ") + "."
}

func buildUnsupportedAnswer(syncs []repository.ConnectorSync) string {
	if len(syncs) == 0 {
		return "I can answer from local source data after you connect a workspace source. Try adding GitHub, Jira, Slack, Google Drive, Notion, SharePoint, or filesystem data."
	}
	return "I can answer local source questions, status questions, and findings requests. Try asking for recent messages, documents, issues, tickets, or source artifacts from a connector."
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
