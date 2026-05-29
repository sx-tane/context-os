package request

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
