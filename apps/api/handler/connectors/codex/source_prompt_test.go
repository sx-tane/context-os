package codex

import (
	"strings"
	"testing"
)

// TestSourceDiscoveryPromptRequestsJiraJQLFirst verifies Jira source discovery prefers JQL over generic Rovo search.
func TestSourceDiscoveryPromptRequestsJiraJQLFirst(t *testing.T) {
	prompt := sourceDiscoveryPrompt("jira")

	for _, want := range []string{
		"Jira JQL issue search tool first",
		"not generic Rovo workspace search",
		"ORDER BY updated DESC",
		"Derive project keys",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("sourceDiscoveryPrompt() missing %q in %q", want, prompt)
		}
	}
}
