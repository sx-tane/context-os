package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"context-os/domain/repository"
)

// answerLiveFanout queries multiple connected live scopes and synthesizes the usable sections.
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

// answerLiveScope runs one live connector lookup and converts the answer into structured sections.
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

// compactLiveError shortens live lookup errors for user-visible fallback messages.
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
