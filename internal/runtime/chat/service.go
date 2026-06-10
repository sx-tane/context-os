package chat

import (
	"context"
	"fmt"
	"strings"

	"context-os/domain/repository"
)

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

// emitProgress forwards non-empty progress lines to the optional caller callback.
func emitProgress(progress func(string), line string) {
	if progress != nil && strings.TrimSpace(line) != "" {
		progress(line)
	}
}

// shouldAskLiveFirst reports whether a concrete source question should try live Codex before local fallback.
func (s *Service) shouldAskLiveFirst(result Result, mode string) bool {
	return s.live != nil &&
		mode != modeLocal &&
		result.SourceURI != "" &&
		result.Connector != "filesystem" &&
		supportsLiveConnector(result.Connector)
}

// normalizeChatMode maps empty or unknown chat modes to auto mode.
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

// isConnectorScopeURI reports whether a source URI is a connector-level broad scope.
func isConnectorScopeURI(connector, sourceURI string) bool {
	return normalizeConnector(connector) != "" &&
		normalizeConnector(connector) == normalizeConnector(sourceURI)
}

// resolveWorkspace finds the workspace from path or ID request scope.
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

// listSyncs returns workspace connector sync state when a sync repository is configured.
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
