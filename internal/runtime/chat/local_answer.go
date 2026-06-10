package chat

import (
	"context"
	"fmt"
	"strings"

	"context-os/domain/repository"
)

// answerArtifacts answers one artifact intent using optional live lookup followed by local artifact fallback.
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
