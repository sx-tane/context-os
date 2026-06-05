// Package chat answers local user questions from workspace-scoped ContextOS repositories.
package chat

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"context-os/domain/repository"
)

const (
	intentArtifacts   = "artifacts"
	intentFindings    = "findings"
	intentStatus      = "status"
	intentUnsupported = "unsupported"
	modeAuto          = "auto"
	modeCodex         = "codex"
	modeLocal         = "local"
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
	sourceSubTokenPat  = regexp.MustCompile(`[-_.]+`)
	githubRepoPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	jiraKeyPattern     = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
)

// Query is one local chat request.
type Query struct {
	WorkspaceID      string
	WorkspacePath    string
	Message          string
	Connector        string
	Connectors       []string
	SourceURI        string
	Mode             string
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
	WorkspaceID      string
	WorkspacePath    string
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

// LiveSessionResetter clears workspace-scoped live chat conversation metadata.
type LiveSessionResetter interface {
	ResetSession(ctx context.Context, workspaceID string) error
}

type liveFanoutResult struct {
	index     int
	connector string
	sourceURI string
	sections  []AnswerSection
	failure   string
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
	query.ResponseLanguage = responseLanguageForMessage(query.ResponseLanguage, message)
	query.Mode = normalizeChatMode(query.Mode)

	workspace, err := s.resolveWorkspace(ctx, query)
	if err != nil {
		return Result{}, err
	}

	syncs, err := s.listSyncs(ctx, workspace.ID)
	if err != nil {
		return Result{}, err
	}

	explicitSourceURI := firstNonEmpty(strings.TrimSpace(query.SourceURI), inferSourceURI(message))
	explicitConnector := normalizeConnector(query.Connector)
	connector := normalizeConnector(firstNonEmpty(explicitConnector, inferConnector(message), inferConnectorFromURI(explicitSourceURI)))
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
	var liveScopes []repository.ConnectorSync
	if s.live != nil && query.Mode != modeLocal {
		liveScopes = liveFanoutScopes(message, query.Connectors, explicitConnector, explicitSourceURI, connector, syncs)
	}
	if len(liveScopes) > 0 {
		intent = intentArtifacts
		connector = "multiple"
		sourceURI = ""
	}

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

	if s.live != nil && query.Mode != modeLocal && len(liveScopes) == 0 && sourceURI == "" && requestsLiveFanout(message, query.Connectors, explicitConnector, explicitSourceURI) {
		emitProgress(query.Progress, "• No concrete connected live source was selected; broad connector scopes are setup-only.")
		result.Intent = intentArtifacts
		result.Answer = buildNoConcreteLiveScopeAnswer(syncs, query.Connectors, query.ResponseLanguage)
		result.Summary = result.Answer
		return result, nil
	}

	switch intent {
	case intentArtifacts:
		if result.Connector == "multiple" {
			return s.answerLiveFanout(ctx, workspace.ID, query, result, message, liveScopes)
		}
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

// ResetSession clears persisted live Codex chat session metadata for one workspace.
func (s *Service) ResetSession(ctx context.Context, query Query) error {
	workspace, err := s.resolveWorkspace(ctx, query)
	if err != nil {
		return err
	}
	resetter, ok := s.live.(LiveSessionResetter)
	if !ok || resetter == nil {
		return nil
	}
	return resetter.ResetSession(ctx, workspace.ID)
}

func (s *Service) answerArtifacts(ctx context.Context, workspaceID string, query Query, result Result, message string) (Result, error) {
	liveFailure := ""
	if s.shouldAskLiveFirst(result, query.Mode) {
		answer, err := s.live.Answer(ctx, LiveQuery{
			WorkspaceID:      result.WorkspaceID,
			WorkspacePath:    result.WorkspacePath,
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
		if query.Mode == modeCodex {
			result.Provider = "codex"
			result.Answer = buildCodexOnlyFailureAnswer(result, liveFailure, query.ResponseLanguage)
			result.Summary = result.Answer
			return result, nil
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

func (s *Service) answerLiveFanout(ctx context.Context, workspaceID string, query Query, result Result, message string, scopes []repository.ConnectorSync) (Result, error) {
	if s.live == nil || len(scopes) == 0 {
		return s.answerArtifacts(ctx, workspaceID, query, result, message)
	}

	resultCh := make(chan liveFanoutResult, len(scopes))
	var wg sync.WaitGroup
	started := 0
	for index, scope := range scopes {
		connector := normalizeConnector(scope.Connector)
		sourceURI := strings.TrimSpace(scope.SourceURI)
		if connector == "" || sourceURI == "" || connector == "filesystem" || !supportsLiveConnector(connector) {
			continue
		}
		started++
		emitProgress(query.Progress, fmt.Sprintf("› Live Codex: %s connected-source lookup", connector))
		wg.Add(1)
		go func(index int, connector, sourceURI string) {
			defer wg.Done()
			answer, err := s.live.Answer(ctx, LiveQuery{
				WorkspaceID:      result.WorkspaceID,
				WorkspacePath:    result.WorkspacePath,
				Connector:        connector,
				SourceURI:        sourceURI,
				Message:          message,
				ResponseLanguage: query.ResponseLanguage,
				Progress:         query.Progress,
			})
			if err != nil {
				failure := fmt.Sprintf("%s: %s", connector, compactLiveError(err))
				resultCh <- liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, failure: failure}
				emitProgress(query.Progress, fmt.Sprintf("• %s live lookup failed: %s; continuing.", connector, compactLiveError(err)))
				return
			}
			answer = strings.TrimSpace(answer)
			if answer == "" {
				resultCh <- liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, failure: fmt.Sprintf("%s: no live answer", connector)}
				return
			}
			liveAnswer, sections := parseLiveAnswer(answer, connector, sourceURI)
			if len(sections) == 0 && liveAnswer != "" {
				sections = []AnswerSection{{
					SourceLabel: connectorDisplayName(connector),
					Connector:   connector,
					SourceURI:   sourceURI,
					Summary:     liveAnswer,
					Confidence:  0.75,
				}}
			}
			resultCh <- liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, sections: sections}
		}(index, connector, sourceURI)
	}
	if started == 0 {
		return s.answerArtifacts(ctx, workspaceID, query, result, message)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]liveFanoutResult, 0, started)
	for liveResult := range resultCh {
		results = append(results, liveResult)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].index < results[j].index
	})

	failures := []string{}
	for i, liveResult := range results {
		if liveResult.failure != "" {
			emitProgress(query.Progress, fmt.Sprintf("• Retrying %s live lookup serially.", liveResult.connector))
			retry := s.answerLiveScope(ctx, query, result, message, liveResult.index, liveResult.connector, liveResult.sourceURI)
			if retry.failure == "" {
				results[i] = retry
				liveResult = retry
			}
		}
		if liveResult.failure != "" {
			failures = append(failures, liveResult.failure)
		} else {
			result.AnswerSections = append(result.AnswerSections, liveResult.sections...)
		}
	}

	if len(result.AnswerSections) > 0 {
		result.Provider = "codex"
		result.Answer = buildFanoutAnswer(result.AnswerSections, failures, query.ResponseLanguage)
		result.Summary = summarizeFanout(result.AnswerSections, query.ResponseLanguage)
		return result, nil
	}
	if query.Mode == modeCodex {
		result.Provider = "codex"
		result.Answer = buildCodexOnlyFailureAnswer(result, strings.Join(failures, "; "), query.ResponseLanguage)
		result.Summary = result.Answer
		return result, nil
	}

	fallback := result
	fallback.Connector = ""
	fallback.SourceURI = ""
	fallback.Provider = "local"
	fallbackResult, err := s.answerArtifacts(ctx, workspaceID, query, fallback, message)
	if err != nil {
		return Result{}, err
	}
	if len(failures) > 0 {
		fallbackResult.Answer = localized(
			query.ResponseLanguage,
			"Connected live sources did not return usable answers.\n\n",
			"已连接的 live sources 没有返回可用答案。\n\n",
			"接続済み live sources から利用できる回答は返りませんでした。\n\n",
			"연결된 live sources에서 사용할 수 있는 답변을 반환하지 않았습니다.\n\n",
		) + fallbackResult.Answer
	}
	return fallbackResult, nil
}

func (s *Service) answerLiveScope(ctx context.Context, query Query, result Result, message string, index int, connector, sourceURI string) liveFanoutResult {
	answer, err := s.live.Answer(ctx, LiveQuery{
		WorkspaceID:      result.WorkspaceID,
		WorkspacePath:    result.WorkspacePath,
		Connector:        connector,
		SourceURI:        sourceURI,
		Message:          message,
		ResponseLanguage: query.ResponseLanguage,
		Progress:         query.Progress,
	})
	if err != nil {
		return liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, failure: fmt.Sprintf("%s: %s", connector, compactLiveError(err))}
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, failure: fmt.Sprintf("%s: no live answer", connector)}
	}
	liveAnswer, sections := parseLiveAnswer(answer, connector, sourceURI)
	if len(sections) == 0 && liveAnswer != "" {
		sections = []AnswerSection{{
			SourceLabel: connectorDisplayName(connector),
			Connector:   connector,
			SourceURI:   sourceURI,
			Summary:     liveAnswer,
			Confidence:  0.75,
		}}
	}
	return liveFanoutResult{index: index, connector: connector, sourceURI: sourceURI, sections: sections}
}

func emitProgress(progress func(string), line string) {
	if progress != nil && strings.TrimSpace(line) != "" {
		progress(line)
	}
}

func (s *Service) shouldAskLiveFirst(result Result, mode string) bool {
	return s.live != nil &&
		mode != modeLocal &&
		result.SourceURI != "" &&
		result.Connector != "filesystem" &&
		supportsLiveConnector(result.Connector)
}

func normalizeChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case modeCodex:
		return modeCodex
	case modeLocal:
		return modeLocal
	default:
		return modeAuto
	}
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

func liveFanoutScopes(message string, requested []string, explicitConnector, explicitSourceURI, inferredConnector string, syncs []repository.ConnectorSync) []repository.ConnectorSync {
	if explicitConnector != "" || explicitSourceURI != "" {
		return nil
	}
	connectors := normalizedConnectorSet(requested)
	if len(connectors) == 0 && inferredConnector != "" {
		return nil
	}
	if len(connectors) == 0 && !shouldAutoFanoutPrompt(message) {
		return nil
	}
	return connectedLiveScopes(syncs, connectors)
}

func requestsLiveFanout(message string, requested []string, explicitConnector, explicitSourceURI string) bool {
	if explicitConnector != "" || explicitSourceURI != "" {
		return false
	}
	return len(normalizedConnectorSet(requested)) > 0
}

func normalizedConnectorSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		connector := normalizeConnector(value)
		if connector == "" || connector == "filesystem" || !supportsLiveConnector(connector) {
			continue
		}
		out[connector] = true
	}
	return out
}

func connectedLiveScopes(syncs []repository.ConnectorSync, allowed map[string]bool) []repository.ConnectorSync {
	concrete := map[string][]repository.ConnectorSync{}
	broad := map[string]repository.ConnectorSync{}
	seen := map[string]bool{}
	for _, sync := range syncs {
		connector := normalizeConnector(sync.Connector)
		sourceURI := strings.TrimSpace(sync.SourceURI)
		if connector == "" || sourceURI == "" || connector == "filesystem" || !supportsLiveConnector(connector) {
			continue
		}
		if len(allowed) > 0 && !allowed[connector] {
			continue
		}
		if !isUsableLiveSync(sync.Status) {
			continue
		}
		key := connector + "\x00" + sourceURI
		if seen[key] {
			continue
		}
		seen[key] = true
		sync.Connector = connector
		sync.SourceURI = sourceURI
		if isConnectorScopeURI(connector, sourceURI) {
			if _, ok := broad[connector]; !ok {
				broad[connector] = sync
			}
			continue
		}
		concrete[connector] = append(concrete[connector], sync)
	}
	order := []string{"jira", "github", "slack", "googledrive", "notion", "sharepoint"}
	out := make([]repository.ConnectorSync, 0, len(syncs))
	for _, connector := range order {
		if len(concrete[connector]) > 0 {
			out = append(out, concrete[connector]...)
			continue
		}
		if scope, ok := broad[connector]; ok {
			out = append(out, scope)
		}
	}
	return out
}

func isUsableLiveSync(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "connected", "idle", "syncing":
		return true
	default:
		return false
	}
}

func shouldAutoFanoutPrompt(message string) bool {
	lower := strings.ToLower(message)
	trimmed := strings.TrimSpace(lower)
	if trimmed == "" {
		return false
	}
	if trimmed == "help" || trimmed == "what can you do" || isLanguageControlPrompt(trimmed) {
		return false
	}
	switch trimmed {
	case "status", "source status", "local source status", "ready", "configured", "synced",
		"findings", "finding", "mismatches", "mismatch", "analyze", "analyse", "analysis":
		return false
	}
	return len([]rune(trimmed)) >= 3
}

func isLanguageControlPrompt(trimmedLower string) bool {
	normalized := strings.Join(strings.Fields(trimmedLower), " ")
	switch normalized {
	case "answer me in english", "respond in english", "use english", "speak english",
		"reply in english", "english please", "please answer in english":
		return true
	}
	return hasAny(normalized, "请用中文回答", "用中文回答", "中文回答", "日本語で答えて", "韓国語で答えて", "한국어로 답변")
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

func buildNoConcreteLiveScopeAnswer(syncs []repository.ConnectorSync, requested []string, language string) string {
	connectors := normalizedConnectorSet(requested)
	if len(connectors) == 0 {
		for _, sync := range syncs {
			connector := normalizeConnector(sync.Connector)
			if connector == "" || connector == "filesystem" || !supportsLiveConnector(connector) {
				continue
			}
			connectors[connector] = true
		}
	}
	names := make([]string, 0, len(connectors))
	for _, connector := range []string{"jira", "github", "slack", "googledrive", "notion", "sharepoint"} {
		if connectors[connector] {
			names = append(names, connectorDisplayName(connector))
		}
	}
	if len(names) == 0 {
		names = append(names, "live connector")
	}
	return fmt.Sprintf(localized(
		language,
		"I did not start Codex because no connected live scope was available for %s. Connect or select a repo, issue, channel, document, folder, or connector scope in Source setup, then ask again.",
		"我没有启动 Codex，因为 %s 没有可用的已连接 live scope。请先在 Source setup 连接或选择 repo、issue、channel、document、folder 或 connector scope，然后再问一次。",
		"Codex は開始しませんでした。%s に利用可能な接続済み live scope がありません。Source setup で repo、issue、channel、document、folder、または connector scope を接続/選択してから再度質問してください。",
		"%s에 사용할 수 있는 연결된 live scope가 없어 Codex를 시작하지 않았습니다. Source setup에서 repo, issue, channel, document, folder 또는 connector scope를 연결하거나 선택한 뒤 다시 질문해 주세요.",
	), strings.Join(names, ", "))
}

func buildCodexOnlyFailureAnswer(result Result, failure, language string) string {
	scope := describeScope(result)
	if strings.TrimSpace(failure) == "" {
		failure = "Codex returned no usable live answer."
	}
	return fmt.Sprintf(localized(
		language,
		"Codex mode is on, but the live lookup for %s did not return a usable answer: %s",
		"Codex 模式已开启，但 %s 的 live 查询没有返回可用答案：%s",
		"Codex mode はオンですが、%s の live lookup から利用できる回答は返りませんでした: %s",
		"Codex mode가 켜져 있지만 %s live 조회에서 사용할 수 있는 답변이 반환되지 않았습니다: %s",
	), scope, failure)
}

func inferSyncSource(message, connector string, syncs []repository.ConnectorSync) repository.ConnectorSync {
	messageTokens := sourceMatchTokens(message)
	normalizedConnector := normalizeConnector(connector)
	var broad repository.ConnectorSync
	for _, sync := range syncs {
		if normalizedConnector != "" && normalizeConnector(sync.Connector) != normalizedConnector {
			continue
		}
		sourceURI := strings.TrimSpace(sync.SourceURI)
		if sourceURI == "" {
			continue
		}
		if isConnectorScopeURI(normalizeConnector(sync.Connector), sourceURI) && broad.SourceURI == "" {
			broad = sync
		}
		for token := range sourceMatchTokens(sourceURI) {
			if messageTokens[token] {
				return sync
			}
		}
	}
	if normalizedConnector != "" {
		return broad
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
		for _, subpart := range sourceSubTokenPat.Split(part, -1) {
			if len(subpart) >= 3 {
				tokens[subpart] = true
			}
		}
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

func buildFanoutAnswer(sections []AnswerSection, failures []string, language string) string {
	answer := fmt.Sprintf(localized(
		language,
		"Found live source context from %d connected source(s).",
		"从 %d 个已连接 source 找到 live context。",
		"%d 件の接続済み source から live context が見つかりました。",
		"%d개의 연결된 source에서 live context를 찾았습니다.",
	), len(sections))
	if synthesis := buildFanoutSynthesis(sections, language); synthesis != "" {
		answer += "\n\n" + synthesis
	}
	if evidence := buildFanoutEvidenceSummary(sections, language); evidence != "" {
		answer += "\n\n" + evidence
	}
	if len(failures) > 0 {
		answer += fmt.Sprintf(localized(
			language,
			"\n\n%d connected source(s) failed or returned no usable answer; see stream for details.",
			"\n\n%d 个已连接 source 失败或没有返回可用答案；详情见 stream。",
			"\n\n%d 件の接続済み source は失敗または利用できる回答なしでした。詳細は stream を確認してください。",
			"\n\n%d개의 연결된 source가 실패했거나 사용할 수 있는 답변을 반환하지 않았습니다. 자세한 내용은 stream을 확인하세요.",
		), len(failures))
	}
	return answer
}

func buildFanoutSynthesis(sections []AnswerSection, language string) string {
	definitions := collectSectionInsights(sections, 2, containsDefinitionSignal)
	behavior := collectSectionInsights(sections, 4, containsBehaviorSignal)
	history := collectSectionInsights(sections, 3, containsHistorySignal)
	openItems := collectOpenItems(sections, 2)
	if len(definitions) == 0 && len(behavior) == 0 && len(history) == 0 && len(openItems) == 0 {
		return ""
	}

	lines := []string{localized(
		language,
		"Summary:",
		"总结：",
		"要約:",
		"요약:",
	)}
	if len(definitions) > 0 {
		lines = append(lines, localized(language, "Meaning: ", "含义：", "意味: ", "의미: ")+strings.Join(definitions, " "))
	}
	if len(behavior) > 0 {
		lines = append(lines, localized(language, "How it works: ", "工作方式：", "動作: ", "동작 방식: ")+strings.Join(behavior, " "))
	}
	if len(history) > 0 {
		lines = append(lines, localized(language, "Change history/current status: ", "变更历史/当前状态：", "変更履歴/現在の状態: ", "변경 이력/현재 상태: ")+strings.Join(history, " "))
	}
	if len(openItems) > 0 {
		lines = append(lines, localized(language, "Open items: ", "待确认事项：", "未確認事項: ", "미확인 항목: ")+strings.Join(openItems, " "))
	}
	return strings.Join(lines, "\n")
}

func buildFanoutEvidenceSummary(sections []AnswerSection, language string) string {
	connectors := map[string]int{}
	for _, section := range sections {
		connector := normalizeConnector(section.Connector)
		if connector == "" {
			continue
		}
		connectors[connector]++
	}
	if len(connectors) == 0 {
		return ""
	}
	parts := make([]string, 0, len(connectors))
	for _, connector := range []string{"jira", "github", "slack", "googledrive", "notion", "sharepoint"} {
		count := connectors[connector]
		if count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", connectorDisplayName(connector), count))
	}
	if len(parts) == 0 {
		return ""
	}
	return localized(
		language,
		"Evidence sources: ",
		"证据来源：",
		"証拠ソース: ",
		"근거 출처: ",
	) + strings.Join(parts, ", ")
}

func collectSectionInsights(sections []AnswerSection, limit int, match func(string) bool) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, section := range sections {
		candidates := section.Facts
		if strings.TrimSpace(section.Summary) != "" {
			candidates = append([]string{section.Summary}, candidates...)
		}
		for _, candidate := range candidates {
			clean := previewText(candidate, 220)
			if clean == "" || seen[clean] || !match(clean) {
				continue
			}
			seen[clean] = true
			out = append(out, ensureTerminalPunctuation(clean))
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func collectOpenItems(sections []AnswerSection, limit int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, section := range sections {
		for _, item := range section.OpenItems {
			clean := previewText(item, 180)
			if clean == "" || seen[clean] {
				continue
			}
			seen[clean] = true
			out = append(out, ensureTerminalPunctuation(clean))
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func containsDefinitionSignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, " means ") ||
		strings.Contains(lower, " is ") ||
		strings.Contains(lower, "defines ") ||
		strings.Contains(lower, "documented as") ||
		strings.Contains(lower, "'0'") ||
		strings.Contains(lower, "'1'")
}

func containsBehaviorSignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "calls ") ||
		strings.Contains(lower, "creates ") ||
		strings.Contains(lower, "query") ||
		strings.Contains(lower, "update") ||
		strings.Contains(lower, "uses ") ||
		strings.Contains(lower, "uploads ") ||
		strings.Contains(lower, "omitted") ||
		strings.Contains(lower, "undefined")
}

func containsHistorySignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "pr #") ||
		strings.Contains(lower, "merged") ||
		strings.Contains(lower, "open pr") ||
		strings.Contains(lower, "proposes") ||
		strings.Contains(lower, "commit ") ||
		strings.Contains(lower, "changed ") ||
		strings.Contains(lower, "recovery")
}

func ensureTerminalPunctuation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	last := value[len(value)-1]
	if last == '.' || last == '!' || last == '?' || last == ':' || last == ';' {
		return value
	}
	return value + "."
}

func summarizeFanout(sections []AnswerSection, language string) string {
	connectors := map[string]struct{}{}
	for _, section := range sections {
		if connector := normalizeConnector(section.Connector); connector != "" {
			connectors[connector] = struct{}{}
		}
	}
	names := make([]string, 0, len(connectors))
	for _, connector := range []string{"jira", "github", "slack", "googledrive", "notion", "sharepoint"} {
		if _, ok := connectors[connector]; ok {
			names = append(names, connectorDisplayName(connector))
		}
	}
	return fmt.Sprintf(localized(
		language,
		"Live source context: %s",
		"Live source context：%s",
		"Live source context: %s",
		"Live source context: %s",
	), strings.Join(names, ", "))
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

func connectorDisplayName(connector string) string {
	switch normalizeConnector(connector) {
	case "github":
		return "GitHub"
	case "jira":
		return "Jira"
	case "slack":
		return "Slack"
	case "googledrive":
		return "Google Drive"
	case "notion":
		return "Notion"
	case "sharepoint":
		return "SharePoint"
	default:
		return connector
	}
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

func responseLanguageForMessage(language, message string) string {
	code := responseLanguageCode(language)
	if code != "zh" {
		return code
	}
	if shouldPreferEnglishForMixedPrompt(message) {
		return "en"
	}
	return code
}

func shouldPreferEnglishForMixedPrompt(message string) bool {
	if containsAnyRange(message, '\uac00', '\ud7af') {
		return false
	}
	if containsAnyRange(message, '\u3040', '\u30ff') {
		return false
	}
	cjkCount := countRunesInRange(message, '\u4e00', '\u9fff')
	if cjkCount == 0 || cjkCount > 6 {
		return false
	}
	if countEnglishWords(message) < 3 {
		return false
	}
	return countChineseCueRunes(message) == 0
}

func containsAnyRange(value string, start, end rune) bool {
	for _, r := range value {
		if r >= start && r <= end {
			return true
		}
	}
	return false
}

func countRunesInRange(value string, start, end rune) int {
	count := 0
	for _, r := range value {
		if r >= start && r <= end {
			count++
		}
	}
	return count
}

func countEnglishWords(value string) int {
	count := 0
	inWord := false
	for _, r := range value {
		isWord := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (inWord && r >= '0' && r <= '9') || (inWord && (r == '_' || r == '-'))
		if isWord && !inWord {
			count++
		}
		inWord = isWord
	}
	return count
}

func countChineseCueRunes(value string) int {
	count := 0
	for _, r := range value {
		switch r {
		case '吗', '呢', '吧', '啊', '的', '了', '是', '有', '和', '在', '请', '问', '中', '文', '回', '答', '最', '近', '变', '化', '什', '么', '怎', '为':
			count++
		}
	}
	return count
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
