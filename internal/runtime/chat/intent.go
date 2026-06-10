package chat

import (
	"strings"
)

// classifyIntent chooses the top-level chat intent from message text and resolved scope.
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
