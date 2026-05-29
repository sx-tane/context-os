// Package request defines inbound JSON request types for the ContextOS API.
package request

// GithubIngest is the JSON body accepted by POST /github/ingest.
type GithubIngest struct {
	URI      string `json:"uri"`
	Token    string `json:"token"`
	Provider string `json:"provider"`
}

// SlackIngest is the JSON body accepted by POST /slack/ingest.
type SlackIngest struct {
	URI      string `json:"uri"`
	Token    string `json:"token"`
	Provider string `json:"provider"`
}

// JiraIngest is the JSON body accepted by POST /jira/ingest.
type JiraIngest struct {
	URI        string            `json:"uri"`
	Content    string            `json:"content"`
	Cursor     string            `json:"cursor"`
	Token      string            `json:"token"`
	Provider   string            `json:"provider"`
	Email      string            `json:"email"`
	APIBaseURL string            `json:"api_base_url"`
	Expand     string            `json:"expand"`
	Metadata   map[string]string `json:"metadata"`
}

// FilesystemIngest is the JSON body accepted by POST /filesystem/ingest for local files or folders.
type FilesystemIngest struct {
	URI      string            `json:"uri"`
	Content  string            `json:"content"`
	Cursor   string            `json:"cursor"`
	Include  string            `json:"include"`
	Exclude  string            `json:"exclude"`
	Metadata map[string]string `json:"metadata"`
}
