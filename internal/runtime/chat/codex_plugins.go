package chat

// supportsLiveConnector reports whether a normalized connector has a live Codex plugin mapping.
func supportsLiveConnector(connector string) bool {
	return livePlugin(connector) != ""
}

// livePlugin maps ContextOS connector names to installed Codex plugin names.
func livePlugin(connector string) string {
	switch normalizeConnector(connector) {
	case "github":
		return "GitHub"
	case "slack":
		return "Slack"
	case "jira":
		return "Atlassian Rovo"
	case "googledrive":
		return "Google Drive"
	case "notion":
		return "Notion"
	case "sharepoint":
		return "SharePoint"
	default:
		return ""
	}
}
