package chat

import (
	"fmt"
	"strings"
)

// livePrompt builds the read-only Codex plugin prompt used for live source-specific chat answers.
func livePrompt(plugin, sourceURI, message, responseLanguage string) string {
	language := normalizeResponseLanguage(responseLanguage)
	jiraRules := ""
	if plugin == "Atlassian Rovo" {
		jiraRules = `
- Before any Jira issue lookup, call Atlassian Rovo's accessible Atlassian resources tool and use the returned cloudId/url. Do not infer or guess a Jira site, Cloud ID, or tenant from prior conversation, project names, or examples.
- For Jira questions, use Atlassian Rovo's Jira JQL issue search tool first, not generic Rovo workspace search. Generic Rovo search can be blocked even when Jira JQL works.
- Build a focused JQL query from the question and source. For issue keys such as BKGDEV-8097, query key = BKGDEV-8097 against each accessible Jira cloudId until found. For connector-wide Jira questions, prefer currentUser(), project keys, issue keys, updated/created/due dates, and ORDER BY updated DESC.
- If generic Rovo search returns "app is not installed on this instance", retry through Jira JQL before declaring Jira unavailable.`
	}
	return fmt.Sprintf(`Use the %s Codex plugin to answer this user question from the live connected account.

Source: %s
Question: %s
Response language: %s

Rules:
- Do not modify any external data.
- Answer in the response language above. If the user mixed languages, prefer the language used for the actual question.
- Start with the direct answer to the user's question and the decision or next action when the evidence supports one.
- Prefer exact source facts over general repository or workspace summaries.
- Use only the %s Codex plugin or context it returns. Do not use gh, git remotes, public web search, or other local/public fallbacks.
- If Source is only a connector name such as github, jira, slack, or googledrive, treat it as the connected account scope for that plugin. Do not inspect or cite unrelated public sources outside the connected account.
- Prefer product-specific read tools inside the selected plugin over broad workspace search when both exist.%s
- Include only the strongest provenance needed to support the answer: source names, links, issue or PR numbers, timestamps, authors, or commit hashes when they materially matter.
- Structure evidence by source so each artifact, thread, issue, PR, or document stays traceable without creating a long inventory.
- Return only JSON with this shape: {"answer":"short plain-text summary","answer_sections":[{"source_label":"human source name","connector":"github|jira|slack|googledrive|notion|sharepoint","source_uri":"exact source URI or key","summary":"short summary","facts":["fact"],"open_items":["open item"],"coding_notes":["coding note"],"links":["https://..."],"timestamps":["timestamp"],"confidence":0.0,"status":"optional status"}]}.
- Use one answer_sections item per real source or artifact. Do not create sections from URL path fragments, enum values, generic terms, or prose tokens.
- Return at most 5 answer_sections unless more are required to avoid a misleading answer.
- In each section, include at most 3 facts and explain why that source changes or supports the answer.
- If multiple activities or thread messages say the same thing, merge them into one concise section and keep the best links.
- If the plugin cannot access the source or the requested fact is unavailable, say that clearly.
- Keep answer concise and readable for chat.`, plugin, sourceURI, strings.TrimSpace(message), language, plugin, jiraRules)
}

// normalizeResponseLanguage converts language codes from clients into explicit prompt language names.
func normalizeResponseLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "zh", "zh-cn", "cn":
		return "Simplified Chinese"
	case "zh-tw", "zh-hant":
		return "Traditional Chinese"
	case "ja", "jp":
		return "Japanese"
	case "ko", "kr":
		return "Korean"
	default:
		return "English"
	}
}
