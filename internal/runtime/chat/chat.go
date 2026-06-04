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
	jiraKeyPattern     = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
)

// Query is one local chat request.
type Query struct {
	WorkspaceID      string
	WorkspacePath    string
	Message          string
	Connector        string
	SourceURI        string
	Timezone         string
	LocalDate        string
	ResponseLanguage string
	Limit            int
	Progress         func(string)
}

// Result is one deterministic local chat answer.
type Result struct {
	Intent         string
	WorkspaceID    string
	WorkspacePath  string
	Connector      string
	SourceURI      string
	Provider       string
	Answer         string
	Summary        string
	AnswerSections []AnswerSection
	Since          *time.Time
	Until          *time.Time
	Artifacts      []repository.IngestEvent
	Syncs          []repository.ConnectorSync
}

// AnswerSection is one structured source-backed section in a chat answer.
type AnswerSection struct {
	SourceLabel string
	Connector   string
	SourceURI   string
	Summary     string
	Facts       []string
	OpenItems   []string
	CodingNotes []string
	Links       []string
	Timestamps  []string
	Confidence  float64
	Status      string
}

// LiveQuery is one optional live source question routed through Codex-backed connectors.
type LiveQuery struct {
	Connector        string
	SourceURI        string
	Message          string
	ResponseLanguage string
	Progress         func(string)
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
		result.Answer = buildStatusAnswer(syncs, query.ResponseLanguage)
		result.Summary = result.Answer
		return result, nil
	case intentFindings:
		result.Answer = localized(
			query.ResponseLanguage,
			"Findings are a local analysis view. Use the findings action to run or refresh mismatch detection, then inspect the evidence and graph in the truth panel.",
			"Findings 是本地分析视图。请使用 findings 操作运行或刷新错配检测，然后在 truth panel 中查看证据和图谱。",
			"Findings はローカル分析ビューです。findings アクションでミスマッチ検出を実行または更新し、truth panel で証拠とグラフを確認してください。",
			"Findings는 로컬 분석 보기입니다. findings 작업으로 불일치 감지를 실행하거나 새로고침한 뒤 truth panel에서 증거와 그래프를 확인하세요.",
		)
		result.Summary = result.Answer
		return result, nil
	default:
		result.Answer = buildUnsupportedAnswer(syncs, query.ResponseLanguage)
		result.Summary = result.Answer
		return result, nil
	}
}

func (s *Service) answerArtifacts(ctx context.Context, workspaceID string, query Query, result Result, message string) (Result, error) {
	liveFailure := ""
	if s.shouldAskLiveFirst(result) {
		answer, err := s.live.Answer(ctx, LiveQuery{
			Connector:        result.Connector,
			SourceURI:        result.SourceURI,
			Message:          message,
			ResponseLanguage: query.ResponseLanguage,
			Progress:         query.Progress,
		})
		if err == nil && strings.TrimSpace(answer) != "" {
			emitProgress(query.Progress, "• Live Codex answer received.")
			result.Provider = "codex"
			result.Answer, result.AnswerSections = parseLiveAnswer(strings.TrimSpace(answer), result.Connector, result.SourceURI)
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
	result.Answer, result.Summary = buildArtifactAnswer(result, limit, query.ResponseLanguage)
	if liveFailure != "" {
		result.Answer = fmt.Sprintf(localized(
			query.ResponseLanguage,
			"Live Codex lookup failed: %s\n\n%s",
			"Live Codex 查询失败：%s\n\n%s",
			"Live Codex の検索に失敗しました: %s\n\n%s",
			"Live Codex 조회에 실패했습니다: %s\n\n%s",
		), liveFailure, result.Answer)
		result.Summary = result.Answer
		return result, nil
	}
	if isCommitQuestion(message) && result.Connector == "github" && !hasCommitArtifact(result.Artifacts) {
		scope := result.SourceURI
		if scope == "" {
			scope = "that GitHub source"
		}
		result.Answer = fmt.Sprintf(localized(
			query.ResponseLanguage,
			"I do not have local commit artifacts for %s yet. Connect Codex-backed GitHub chat or ingest commit data to answer this directly.\n\n%s",
			"我还没有 %s 的本地 commit 证据。请连接 Codex 支持的 GitHub chat，或先摄取 commit 数据后再直接回答。\n\n%s",
			"%s のローカル commit アーティファクトはまだありません。直接答えるには Codex 対応の GitHub chat を接続するか、commit データを取り込んでください。\n\n%s",
			"아직 %s에 대한 로컬 commit 아티팩트가 없습니다. 직접 답하려면 Codex 기반 GitHub chat을 연결하거나 commit 데이터를 먼저 수집하세요.\n\n%s",
		), scope, result.Answer)
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
	if jiraKeyPattern.MatchString(message) {
		return "jira"
	}
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
	if trimmed := strings.Trim(match, `.,;:"'()[]{} `); trimmed != "" {
		return trimmed
	}
	if match := jiraKeyPattern.FindString(message); match != "" {
		return match
	}
	return ""
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

func buildArtifactAnswer(result Result, limit int, language string) (string, string) {
	description := describeScope(result)
	if len(result.Artifacts) == 0 {
		answer := fmt.Sprintf(localized(
			language,
			"No local %s artifacts were found. Connect or sync that source, then ask again.",
			"没有找到本地 %s 证据。请先连接或同步该来源，然后再试。",
			"ローカルの %s アーティファクトは見つかりませんでした。先にそのソースを接続または同期してから、もう一度質問してください。",
			"로컬 %s 아티팩트를 찾지 못했습니다. 먼저 해당 소스를 연결하거나 동기화한 뒤 다시 질문해 주세요.",
		), description)
		return answer, answer
	}

	latest := compactArtifactTitle(result.Artifacts[0])
	answer := fmt.Sprintf(localized(
		language,
		"Found %d local %s artifacts. Latest: %s",
		"找到 %d 条本地 %s 证据。最新：%s",
		"%d 件のローカル %s アーティファクトが見つかりました。最新: %s",
		"%d개의 로컬 %s 아티팩트를 찾았습니다. 최신: %s",
	), len(result.Artifacts), description, latest)
	if len(result.Artifacts) == limit {
		answer += fmt.Sprintf(localized(
			language,
			" Showing the latest %d results.",
			" 显示最新 %d 条结果。",
			" 最新 %d 件の結果を表示しています。",
			" 최신 %d개 결과를 표시합니다.",
		), limit)
	}
	if highlights := compactHighlights(result.Artifacts); len(highlights) > 0 {
		answer += localized(language, "\n\nKey points:", "\n\n要点：", "\n\n要点:", "\n\n핵심 내용:")
		for _, highlight := range highlights {
			answer += "\n- " + highlight
		}
		answer += localized(
			language,
			"\n\nOpen evidence for the full source text.",
			"\n\n打开 evidence 可查看完整来源文本。",
			"\n\n完全なソース本文は evidence を開いて確認してください。",
			"\n\n전체 소스 텍스트는 evidence를 열어 확인하세요.",
		)
	}
	return answer, summarizeArtifacts(result.Artifacts, language)
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

func summarizeArtifacts(artifacts []repository.IngestEvent, language string) string {
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
	return localized(
		language,
		"Latest local artifacts: ",
		"最新本地证据：",
		"最新のローカルアーティファクト: ",
		"최신 로컬 아티팩트: ",
	) + strings.Join(items, " | ")
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

func buildStatusAnswer(syncs []repository.ConnectorSync, language string) string {
	if len(syncs) == 0 {
		return localized(
			language,
			"No local sources are configured for this workspace yet. Add a source to build the local truth store.",
			"这个 workspace 还没有配置本地来源。请添加来源来建立本地 truth store。",
			"この workspace にはまだローカルソースが設定されていません。ローカル truth store を作るにはソースを追加してください。",
			"이 workspace에는 아직 로컬 소스가 설정되어 있지 않습니다. 로컬 truth store를 만들려면 소스를 추가하세요.",
		)
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
	return localized(
		language,
		"Local source status: ",
		"本地来源状态：",
		"ローカルソース状態: ",
		"로컬 소스 상태: ",
	) + strings.Join(parts, ", ") + "."
}

func buildUnsupportedAnswer(syncs []repository.ConnectorSync, language string) string {
	if len(syncs) == 0 {
		return localized(
			language,
			"I can answer from local source data after you connect a workspace source. Try adding GitHub, Jira, Slack, Google Drive, Notion, SharePoint, or filesystem data.",
			"连接 workspace 来源后，我可以基于本地来源数据回答。可以先添加 GitHub、Jira、Slack、Google Drive、Notion、SharePoint 或 filesystem 数据。",
			"workspace のソースを接続すると、ローカルソースデータから回答できます。GitHub、Jira、Slack、Google Drive、Notion、SharePoint、または filesystem データを追加してください。",
			"workspace 소스를 연결하면 로컬 소스 데이터에서 답변할 수 있습니다. GitHub, Jira, Slack, Google Drive, Notion, SharePoint 또는 filesystem 데이터를 추가해 보세요.",
		)
	}
	return localized(
		language,
		"I can answer local source questions, status questions, and findings requests. Try asking for recent messages, documents, issues, tickets, or source artifacts from a connector.",
		"我可以回答本地来源问题、状态问题和 findings 请求。你可以询问某个 connector 的近期消息、文档、issue、ticket 或 source artifact。",
		"ローカルソースの質問、ステータスの質問、findings リクエストに回答できます。connector の最近のメッセージ、ドキュメント、issue、ticket、source artifact について聞いてみてください。",
		"로컬 소스 질문, 상태 질문, findings 요청에 답할 수 있습니다. connector의 최근 메시지, 문서, issue, ticket 또는 source artifact에 대해 물어보세요.",
	)
}

func localized(language, english, simplifiedChinese, japanese, korean string) string {
	switch responseLanguageCode(language) {
	case "zh":
		return simplifiedChinese
	case "ja":
		return japanese
	case "ko":
		return korean
	default:
		return english
	}
}

func responseLanguageCode(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "zh", "zh-cn", "cn", "zh-tw", "zh-hant":
		return "zh"
	case "ja", "jp":
		return "ja"
	case "ko", "kr":
		return "ko"
	default:
		return "en"
	}
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
