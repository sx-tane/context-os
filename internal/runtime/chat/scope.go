package chat

import (
	"strings"

	"context-os/domain/repository"
)

// liveFanoutScopes selects connected source scopes for broad multi-connector live lookup.
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

// requestsLiveFanout reports whether the request should use connected live source fanout.
func requestsLiveFanout(message string, requested []string, explicitConnector, explicitSourceURI string) bool {
	if explicitConnector != "" || explicitSourceURI != "" {
		return false
	}
	return len(normalizedConnectorSet(requested)) > 0
}

// normalizedConnectorSet normalizes requested connector names into a lookup set.
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

// connectedLiveScopes returns usable non-filesystem sync records in stable connector order.
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

// isUsableLiveSync reports whether a connector sync status can be queried live.
func isUsableLiveSync(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "connected", "idle", "syncing":
		return true
	default:
		return false
	}
}

// shouldAutoFanoutPrompt detects meaningful broad prompts that should search connected live sources.
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

// isLanguageControlPrompt detects prompts that only control response language and should not fan out.
func isLanguageControlPrompt(trimmedLower string) bool {
	normalized := strings.Join(strings.Fields(trimmedLower), " ")
	switch normalized {
	case "answer me in english", "respond in english", "use english", "speak english",
		"reply in english", "english please", "please answer in english":
		return true
	}
	return hasAny(normalized, "请用中文回答", "用中文回答", "中文回答", "日本語で答えて", "韓国語で答えて", "한국어로 답변")
}

// inferSyncSource matches user text to a saved connector sync source.
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

// sourceMatchTokens builds normalized matching tokens for a source label or URI.
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
