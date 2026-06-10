package chat

import (
	"fmt"
	"sort"
	"strings"

	"context-os/domain/repository"
)

// buildArtifactAnswer renders local artifact query results into an answer and summary.
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

// describeScope summarizes the connector, source, and date range used by an artifact answer.
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

// buildNoConcreteLiveScopeAnswer explains that broad connected-account setup lacks a concrete live source.
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

// buildCodexOnlyFailureAnswer renders a Codex-mode failure without local fallback.
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

// isCommitQuestion detects GitHub commit prompts that need commit-level evidence.
func isCommitQuestion(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "commit")
}

// hasCommitArtifact reports whether local artifacts contain commit evidence.
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

// summarizeArtifacts builds the compatibility summary for local artifact answers.
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

// buildFanoutAnswer renders the plain answer for multi-source live fanout sections.
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

// buildFanoutSynthesis synthesizes meaning, behavior, status, and open-item lines from sections.
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

// buildFanoutEvidenceSummary summarizes how many sources contributed live evidence.
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

// collectSectionInsights collects section facts that match a synthesis signal predicate.
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

// collectOpenItems collects unresolved items from fanout sections.
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

// containsDefinitionSignal detects facts that explain meaning or definitions.
func containsDefinitionSignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, " means ") ||
		strings.Contains(lower, " is ") ||
		strings.Contains(lower, "defines ") ||
		strings.Contains(lower, "documented as") ||
		strings.Contains(lower, "'0'") ||
		strings.Contains(lower, "'1'")
}

// containsBehaviorSignal detects facts that explain behavior or implementation flow.
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

// containsHistorySignal detects facts that explain change, status, or timing.
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

// ensureTerminalPunctuation makes synthesized answer lines read as complete sentences.
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

// summarizeFanout builds a short source-count summary for live fanout answers.
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

// compactArtifactTitle returns a short display title for one stored artifact.
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

// compactHighlights extracts concise highlights from local artifacts for answer bullets.
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

// candidateHighlightLines splits artifact body text into candidate highlight lines.
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

// cleanHighlightLine trims markup and bullets from a candidate highlight.
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

// previewText truncates text without splitting the desired display limit badly.
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

// connectorDisplayName returns the human-facing connector label.
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

// buildStatusAnswer summarizes connected local source status counts.
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

// buildUnsupportedAnswer explains which chat requests the local service can answer.
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
