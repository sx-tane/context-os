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

	sourcePattern      = regexp.MustCompile(`(?i)(#[A-Za-z0-9_.-]+|[a-z]+://[^\s,]+|https?://[^\s,]+|[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`)
	sourceTokenPattern = regexp.MustCompile(`[^a-z0-9_.-]+`)
	githubRepoPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
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
	Progress      func(string)
}

// Result is one deterministic local chat answer.
type Result struct {
	Intent        string
	WorkspaceID   string
	WorkspacePath string
	Connector     string
	SourceURI     string
	Provider      string
	Answer        string
	Summary       string
	Since         *time.Time
	Until         *time.Time
	Artifacts     []repository.IngestEvent
	Syncs         []repository.ConnectorSync
}

// LiveQuery is one optional live source question routed through Codex-backed connectors.
type LiveQuery struct {
	Connector string
	SourceURI string
	Message   string
	Progress  func(string)
}

// LiveAnswerer answers a source question from a live connector account.
type LiveAnswerer interface {
	Answer(ctx context.Context, query LiveQuery) (string, error)
}

// Service routes local chat queries to workspace repositories.
type Service struct {
	workspaces repository.WorkspaceRepository
	events     repository.EventRepository
	syncs      repository.SyncRepository
	live       LiveAnswerer
}

// NewService returns a Service backed by the provided repositories.
func NewService(workspaces repository.WorkspaceRepository, events repository.EventRepository, syncs repository.SyncRepository) *Service {
	return &Service{workspaces: workspaces, events: events, syncs: syncs}
}

// NewServiceWithLiveAnswerer returns a Service that can optionally ask live Codex-backed sources.
func NewServiceWithLiveAnswerer(workspaces repository.WorkspaceRepository, events repository.EventRepository, syncs repository.SyncRepository, live LiveAnswerer) *Service {
	return &Service{workspaces: workspaces, events: events, syncs: syncs, live: live}
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

	explicitSourceURI := firstNonEmpty(strings.TrimSpace(query.SourceURI), inferSourceURI(message))
	connector := normalizeConnector(firstNonEmpty(query.Connector, inferConnector(message), inferConnectorFromURI(explicitSourceURI)))
	sourceURI := explicitSourceURI
	if sourceURI == "" {
		if match := inferSyncSource(message, connector, syncs); match.SourceURI != "" {
			sourceURI = match.SourceURI
			if connector == "" {
				connector = normalizeConnector(match.Connector)
			}
		}
	}
	since, until := inferTimeRange(query, message)
	intent := classifyIntent(message, connector, sourceURI)

	result := Result{
		Intent:        intent,
		WorkspaceID:   workspace.ID,
		WorkspacePath: workspace.Path,
		Connector:     connector,
		SourceURI:     sourceURI,
		Provider:      "local",
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
	liveFailure := ""
	if s.shouldAskLiveFirst(result) {
		answer, err := s.live.Answer(ctx, LiveQuery{
			Connector: result.Connector,
			SourceURI: result.SourceURI,
			Message:   message,
			Progress:  query.Progress,
		})
		if err == nil && strings.TrimSpace(answer) != "" {
			emitProgress(query.Progress, "• Live Codex answer received.")
			result.Provider = "codex"
			result.Answer = strings.TrimSpace(answer)
			result.Summary = result.Answer
			return result, nil
		}
		if err != nil {
			liveFailure = compactLiveError(err)
		} else {
			liveFailure = "Codex returned no live answer."
		}
		emitProgress(query.Progress, "• Live Codex did not complete; checking Local DB fallback.")
	}

	limit := clampLimit(query.Limit)
	localSourceURI := result.SourceURI
	if isConnectorScopeURI(result.Connector, result.SourceURI) {
		localSourceURI = ""
	}
	eventQuery := repository.EventQuery{
		Connector: result.Connector,
		SourceURI: localSourceURI,
		Text:      inferSearchText(message),
		Since:     result.Since,
		Until:     result.Until,
		Limit:     limit,
	}

	artifacts, err := s.events.Query(ctx, workspaceID, eventQuery)
	if err != nil {
		return Result{}, fmt.Errorf("chat: query artifacts: %w", err)
	}
	emitProgress(query.Progress, fmt.Sprintf("• Local DB returned %d artifact(s).", len(artifacts)))
	result.Artifacts = artifacts
	result.Answer, result.Summary = buildArtifactAnswer(result, limit)
	if liveFailure != "" {
		result.Answer = fmt.Sprintf("Live Codex lookup failed: %s\n\n%s", liveFailure, result.Answer)
		result.Summary = result.Answer
		return result, nil
	}
	if isCommitQuestion(message) && result.Connector == "github" && !hasCommitArtifact(result.Artifacts) {
		scope := result.SourceURI
		if scope == "" {
			scope = "that GitHub source"
		}
		result.Answer = fmt.Sprintf("I do not have local commit artifacts for %s yet. Connect Codex-backed GitHub chat or ingest commit data to answer this directly.\n\n%s", scope, result.Answer)
		result.Summary = result.Answer
	}
	return result, nil
}

func emitProgress(progress func(string), line string) {
	if progress != nil && strings.TrimSpace(line) != "" {
		progress(line)
	}
}

func (s *Service) shouldAskLiveFirst(result Result) bool {
	return s.live != nil &&
		result.SourceURI != "" &&
		result.Connector != "filesystem" &&
		supportsLiveConnector(result.Connector)
}

func isConnectorScopeURI(connector, sourceURI string) bool {
	return normalizeConnector(connector) != "" &&
		normalizeConnector(connector) == normalizeConnector(sourceURI)
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

func inferConnectorFromURI(uri string) string {
	lower := strings.ToLower(strings.TrimSpace(uri))
	switch {
	case lower == "":
		return ""
	case strings.HasPrefix(lower, "#"), strings.HasPrefix(lower, "slack://"), strings.Contains(lower, "slack.com"):
		return "slack"
	case strings.HasPrefix(lower, "github://"), strings.Contains(lower, "github.com"), strings.Contains(lower, "api.github.com"):
		return "github"
	case githubRepoPattern.MatchString(strings.TrimSpace(uri)):
		return "github"
	case strings.HasPrefix(lower, "jira://"), strings.Contains(lower, "atlassian.net"), strings.Contains(lower, "/browse/"):
		return "jira"
	case strings.HasPrefix(lower, "notion://"), strings.Contains(lower, "notion.so"), strings.Contains(lower, "notion.site"):
		return "notion"
	case strings.HasPrefix(lower, "googledrive://"), strings.HasPrefix(lower, "gdrive://"), strings.Contains(lower, "drive.google.com"), strings.Contains(lower, "docs.google.com"):
		return "googledrive"
	case strings.HasPrefix(lower, "sharepoint://"), strings.Contains(lower, "sharepoint.com"), strings.Contains(lower, "onedrive.live.com"):
		return "sharepoint"
	default:
		return ""
	}
}

func normalizeConnector(connector string) string {
	connector = strings.ToLower(strings.TrimSpace(connector))
	connector = strings.ReplaceAll(connector, "google-drive", "googledrive")
	connector = strings.ReplaceAll(connector, "google_drive", "googledrive")
	connector = strings.ReplaceAll(connector, "google drive", "googledrive")
	connector = strings.ReplaceAll(connector, "gdrive", "googledrive")
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

	latest := compactArtifactTitle(result.Artifacts[0])
	answer := fmt.Sprintf("Found %d local %s artifacts. Latest: %s", len(result.Artifacts), description, latest)
	if len(result.Artifacts) == limit {
		answer += fmt.Sprintf(" Showing the latest %d results.", limit)
	}
	if highlights := compactHighlights(result.Artifacts); len(highlights) > 0 {
		answer += "\n\nKey points:"
		for _, highlight := range highlights {
			answer += "\n- " + highlight
		}
		answer += "\n\nOpen evidence for the full source text."
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

func inferSyncSource(message, connector string, syncs []repository.ConnectorSync) repository.ConnectorSync {
	messageTokens := sourceMatchTokens(message)
	normalizedConnector := normalizeConnector(connector)
	for _, sync := range syncs {
		if normalizedConnector != "" && normalizeConnector(sync.Connector) != normalizedConnector {
			continue
		}
		sourceURI := strings.TrimSpace(sync.SourceURI)
		if sourceURI == "" {
			continue
		}
		for token := range sourceMatchTokens(sourceURI) {
			if messageTokens[token] {
				return sync
			}
		}
	}
	if normalizedConnector == "" {
		return repository.ConnectorSync{}
	}
	for _, sync := range syncs {
		if normalizeConnector(sync.Connector) != normalizedConnector {
			continue
		}
		if isConnectorScopeURI(normalizedConnector, sync.SourceURI) {
			return sync
		}
	}
	return repository.ConnectorSync{}
}

func sourceMatchTokens(value string) map[string]bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "https://github.com/")
	normalized = strings.TrimPrefix(normalized, "http://github.com/")
	normalized = strings.TrimPrefix(normalized, "https://api.github.com/repos/")
	normalized = strings.TrimPrefix(normalized, "http://api.github.com/repos/")
	normalized = strings.TrimPrefix(normalized, "github://repos/")
	normalized = strings.TrimPrefix(normalized, "github://")
	normalized = strings.TrimPrefix(normalized, "repo://")
	normalized = strings.Trim(normalized, `.,;:"'()[]{} `)

	parts := sourceTokenPattern.Split(normalized, -1)
	tokens := map[string]bool{}
	for _, part := range parts {
		part = strings.Trim(part, "-_.")
		if len(part) < 3 {
			continue
		}
		tokens[part] = true
	}
	if len(parts) >= 2 {
		owner := strings.Trim(parts[0], "-_.")
		repo := strings.Trim(parts[1], "-_.")
		if owner != "" && repo != "" {
			tokens[owner+"/"+repo] = true
		}
	}
	return tokens
}

func isCommitQuestion(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "commit")
}

func hasCommitArtifact(artifacts []repository.IngestEvent) bool {
	for _, artifact := range artifacts {
		if strings.EqualFold(artifact.Metadata["object_type"], "commit") {
			return true
		}
		if strings.Contains(strings.ToLower(artifact.SourceURI), "/commit/") {
			return true
		}
	}
	return false
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

func compactArtifactTitle(artifact repository.IngestEvent) string {
	title := strings.TrimSpace(artifact.Title)
	if title == "" || len(strings.Fields(title)) > 18 {
		title = strings.TrimSpace(artifact.SourceURI)
	}
	if title == "" {
		title = artifact.Body
	}
	return previewText(title, 90)
}

func compactHighlights(artifacts []repository.IngestEvent) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		for _, line := range candidateHighlightLines(artifact.Body) {
			line = cleanHighlightLine(line)
			if line == "" {
				continue
			}
			key := strings.ToLower(line)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, previewText(line, 150))
			if len(out) >= 6 {
				return out
			}
		}
	}
	return out
}

func candidateHighlightLines(body string) []string {
	lines := strings.Split(body, "\n")
	out := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			out = append(out, trimmed)
			continue
		}
		if hasAny(lower, "todo", "asked", "review", "release", "hotfix", "blocked", "confirmed", "requested", "pr:", "issue:") {
			out = append(out, trimmed)
		}
	}
	return out
}

func cleanHighlightLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "**", "")
	line = strings.ReplaceAll(line, "`", "")
	if strings.HasPrefix(strings.ToLower(line), "source:") ||
		strings.HasPrefix(strings.ToLower(line), "message link:") ||
		strings.HasPrefix(strings.ToLower(line), "channel link:") {
		return ""
	}
	return line
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

func compactLiveError(err error) string {
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "unknown error"
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return previewText(line, 180)
		}
	}
	return previewText(text, 180)
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
