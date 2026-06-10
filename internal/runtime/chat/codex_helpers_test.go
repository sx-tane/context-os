package chat

// These white-box tests cover package-local helpers extracted from the Codex answerer split.

import "testing"

// TestLivePluginMapsSupportedConnectors verifies ContextOS connector names map to installed Codex plugin names.
func TestLivePluginMapsSupportedConnectors(t *testing.T) {
	cases := []struct {
		connector string
		want      string
	}{
		{connector: "github", want: "GitHub"},
		{connector: "slack", want: "Slack"},
		{connector: "jira", want: "Atlassian Rovo"},
		{connector: "googledrive", want: "Google Drive"},
		{connector: "notion", want: "Notion"},
		{connector: "sharepoint", want: "SharePoint"},
	}

	for _, tc := range cases {
		t.Run(tc.connector, func(t *testing.T) {
			if got := livePlugin(tc.connector); got != tc.want {
				t.Errorf("livePlugin(%q) = %q, want %q", tc.connector, got, tc.want)
			}
			if !supportsLiveConnector(tc.connector) {
				t.Errorf("supportsLiveConnector(%q) = false, want true", tc.connector)
			}
		})
	}
}

// TestCodexSessionHelpersKeepStableKeys verifies session key versioning and safe filenames stay stable.
func TestCodexSessionHelpersKeepStableKeys(t *testing.T) {
	if got := codexSessionKey("workspace", "jira"); got != "workspace::jira-jql-v2" {
		t.Fatalf("codexSessionKey() = %q, want %q", got, "workspace::jira-jql-v2")
	}
	if got := codexSessionKey("workspace", "github"); got != "workspace::github" {
		t.Fatalf("codexSessionKey() = %q, want %q", got, "workspace::github")
	}
	if got := safeSessionFilename(" ../workspace::github!! "); got != "workspace_github" {
		t.Fatalf("safeSessionFilename() = %q, want %q", got, "workspace_github")
	}
}

// TestNormalizeResponseLanguageUsesPromptNames verifies client language codes become explicit prompt language names.
func TestNormalizeResponseLanguageUsesPromptNames(t *testing.T) {
	cases := []struct {
		language string
		want     string
	}{
		{language: "zh", want: "Simplified Chinese"},
		{language: "zh-tw", want: "Traditional Chinese"},
		{language: "ja", want: "Japanese"},
		{language: "ko", want: "Korean"},
		{language: "", want: "English"},
	}

	for _, tc := range cases {
		t.Run(tc.language, func(t *testing.T) {
			if got := normalizeResponseLanguage(tc.language); got != tc.want {
				t.Errorf("normalizeResponseLanguage(%q) = %q, want %q", tc.language, got, tc.want)
			}
		})
	}
}

// TestIsResumeSessionUnavailableRecognizesMissingSessionText verifies archived or missing Codex sessions trigger a fresh session.
func TestIsResumeSessionUnavailableRecognizesMissingSessionText(t *testing.T) {
	if !isResumeSessionUnavailable("session archived", nilError("resume failed")) {
		t.Fatalf("isResumeSessionUnavailable() = false, want true")
	}
	if isResumeSessionUnavailable("rate limited", nilError("temporary failure")) {
		t.Fatalf("isResumeSessionUnavailable() = true, want false")
	}
}

type nilError string

func (e nilError) Error() string { return string(e) }
