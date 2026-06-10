package chat

// White-box tests cover package-local inference helpers split out of chat.go.

import (
	"testing"
	"time"
)

// TestInferConnectorDetectsSupportedConnectorWords verifies connector names and aliases normalize to canonical connector IDs.
func TestInferConnectorDetectsSupportedConnectorWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "github", message: "show github commits", want: "github"},
		{name: "google drive alias", message: "check google drive docs", want: "googledrive"},
		{name: "jira", message: "what changed in jira", want: "jira"},
		{name: "unknown", message: "what changed recently", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := inferConnector(tt.message); got != tt.want {
				t.Errorf("inferConnector(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

// TestInferConnectorFromURIMapsKnownSourceFormats verifies pasted source URIs route to the matching connector.
func TestInferConnectorFromURIMapsKnownSourceFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "github repo", uri: "owner/repo", want: "github"},
		{name: "jira url", uri: "https://team.atlassian.net/browse/BKGDEV-123", want: "jira"},
		{name: "slack", uri: "slack://C123/p456", want: "slack"},
		{name: "notion", uri: "https://www.notion.so/team/Page-0123456789abcdef0123456789abcdef", want: "notion"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := inferConnectorFromURI(tt.uri); got != tt.want {
				t.Errorf("inferConnectorFromURI(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

// TestInferSourceURIExtractsExplicitSources verifies source extraction prefers links, channels, repo slugs, and Jira keys.
func TestInferSourceURIExtractsExplicitSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "channel", message: "summarize #team today", want: "#team"},
		{name: "url", message: "read https://example.test/doc, please", want: "https://example.test/doc"},
		{name: "repo", message: "what changed in owner/repo?", want: "owner/repo"},
		{name: "jira key", message: "what is BKGDEV-123 doing", want: "BKGDEV-123"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := inferSourceURI(tt.message); got != tt.want {
				t.Errorf("inferSourceURI(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

// TestInferTimeRangeUsesExplicitLocalDate verifies today prompts use the request local date as the UTC day range.
func TestInferTimeRangeUsesExplicitLocalDate(t *testing.T) {
	t.Parallel()

	since, until := inferTimeRange(Query{LocalDate: "2026-06-10", Timezone: "UTC"}, "show slack today")
	if since == nil || until == nil {
		t.Fatalf("inferTimeRange() = %v, %v, want both bounds", since, until)
	}
	if got := since.Format(time.RFC3339); got != "2026-06-10T00:00:00Z" {
		t.Fatalf("since = %q, want 2026-06-10T00:00:00Z", got)
	}
	if got := until.Format(time.RFC3339); got != "2026-06-11T00:00:00Z" {
		t.Fatalf("until = %q, want 2026-06-11T00:00:00Z", got)
	}
}
