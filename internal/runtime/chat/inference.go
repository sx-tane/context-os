package chat

import (
	"strings"
	"time"
)

// inferConnector detects a connector name mentioned in the user message.
func inferConnector(message string) string {
	lower := strings.ToLower(message)
	if jiraKeyPattern.MatchString(message) {
		return "jira"
	}
	checks := []struct {
		name  string
		terms []string
	}{
		{name: "googledrive", terms: []string{"google drive", "googledrive", "gdrive"}},
		{name: "sharepoint", terms: []string{"sharepoint", "one drive", "onedrive"}},
		{name: "github", terms: []string{"github", "pull request", "pr ", "repo", "commit"}},
		{name: "slack", terms: []string{"slack", "channel"}},
		{name: "jira", terms: []string{"jira", "ticket", "issue", "sprint"}},
		{name: "notion", terms: []string{"notion"}},
		{name: "filesystem", terms: []string{"filesystem", "file system", "local file", "docs/"}},
	}
	for _, check := range checks {
		for _, term := range check.terms {
			if strings.Contains(lower, term) {
				return check.name
			}
		}
	}
	return ""
}

// inferConnectorFromURI maps pasted source URIs to their owning connector.
func inferConnectorFromURI(uri string) string {
	lower := strings.ToLower(strings.TrimSpace(uri))
	switch {
	case lower == "":
		return ""
	case strings.HasPrefix(lower, "#"), strings.HasPrefix(lower, "slack://"), strings.Contains(lower, "slack.com"):
		return "slack"
	case strings.HasPrefix(lower, "github://"), strings.Contains(lower, "github.com"), strings.Contains(lower, "api.github.com"):
		return "github"
	case githubRepoPattern.MatchString(strings.TrimSpace(uri)):
		return "github"
	case strings.HasPrefix(lower, "jira://"), strings.Contains(lower, "atlassian.net"), strings.Contains(lower, "/browse/"):
		return "jira"
	case strings.HasPrefix(lower, "notion://"), strings.Contains(lower, "notion.so"), strings.Contains(lower, "notion.site"):
		return "notion"
	case strings.HasPrefix(lower, "googledrive://"), strings.HasPrefix(lower, "gdrive://"), strings.Contains(lower, "drive.google.com"), strings.Contains(lower, "docs.google.com"):
		return "googledrive"
	case strings.HasPrefix(lower, "sharepoint://"), strings.Contains(lower, "sharepoint.com"), strings.Contains(lower, "onedrive.live.com"):
		return "sharepoint"
	default:
		return ""
	}
}

// normalizeConnector canonicalizes connector aliases used by requests and messages.
func normalizeConnector(connector string) string {
	connector = strings.ToLower(strings.TrimSpace(connector))
	connector = strings.ReplaceAll(connector, "google-drive", "googledrive")
	connector = strings.ReplaceAll(connector, "google_drive", "googledrive")
	connector = strings.ReplaceAll(connector, "google drive", "googledrive")
	connector = strings.ReplaceAll(connector, "gdrive", "googledrive")
	return connector
}

// inferSourceURI extracts an explicit source URI, channel, repo slug, or Jira key from the message.
func inferSourceURI(message string) string {
	match := sourcePattern.FindString(message)
	if trimmed := strings.Trim(match, `.,;:"'()[]{} `); trimmed != "" {
		return trimmed
	}
	if match := jiraKeyPattern.FindString(message); match != "" {
		return match
	}
	return ""
}

// inferSearchText keeps user text for local artifact filtering when it is not just routing text.
func inferSearchText(message string) string {
	lower := strings.ToLower(message)
	for _, marker := range []string{"containing ", "contains ", "mentioning ", "mentions ", "about "} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		text := strings.TrimSpace(message[idx+len(marker):])
		text = strings.Trim(text, `.,;:"'()[]{} `)
		if len(text) > 2 && !hasAny(strings.ToLower(text), " it", "this", "that") {
			return text
		}
	}
	return ""
}

// inferTimeRange converts explicit or natural date hints into an artifact query range.
func inferTimeRange(query Query, message string) (*time.Time, *time.Time) {
	lower := strings.ToLower(message)
	location := loadLocation(query.Timezone)
	base := localDate(query.LocalDate, location)

	if strings.Contains(lower, "yesterday") {
		start := time.Date(base.Year(), base.Month(), base.Day()-1, 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	if strings.Contains(lower, "today") {
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	if strings.Contains(lower, "this week") || strings.Contains(lower, "week") {
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -6)
		end := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
		return utcPtr(start), utcPtr(end)
	}
	return nil, nil
}

// loadLocation returns the requested timezone or the process local timezone when it cannot be loaded.
func loadLocation(name string) *time.Location {
	if strings.TrimSpace(name) == "" {
		return time.Local
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return location
}

// localDate parses a local YYYY-MM-DD date and falls back to today in the chosen timezone.
func localDate(raw string, location *time.Location) time.Time {
	if raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, location)
		if err == nil {
			return parsed
		}
	}
	return time.Now().In(location)
}

// utcPtr converts a time to UTC and returns it by pointer for query filters.
func utcPtr(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}
