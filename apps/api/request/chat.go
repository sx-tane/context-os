package request

// ChatQuery is the JSON body accepted by POST /chat/query.
type ChatQuery struct {
	// WorkspaceID is the selected workspace path or stored workspace identifier.
	WorkspaceID string `json:"workspace_id" example:"/home/user/myproject"`
	// WorkspacePath is an optional explicit workspace path; it takes precedence over WorkspaceID.
	WorkspacePath string `json:"workspace_path" example:"/home/user/myproject"`
	// Message is the user question to answer from local ContextOS data.
	Message string `json:"message" example:"give me today's Slack messages"`
	// Connector optionally pins the query to a connector such as slack, github, jira, or filesystem.
	Connector string `json:"connector" example:"slack"`
	// Connectors optionally pins the query to several live connectors.
	Connectors []string `json:"connectors,omitempty" example:"jira,github,slack"`
	// SourceURI optionally pins the query to a channel, repository, folder, or document URI.
	SourceURI string `json:"source_uri" example:"#delivery-team"`
	// Timezone is the user's IANA timezone, used for local date words such as today.
	Timezone string `json:"timezone" example:"Asia/Kuala_Lumpur"`
	// LocalDate is the user's current local date in YYYY-MM-DD form.
	LocalDate string `json:"local_date" example:"2026-06-01"`
	// ResponseLanguage is a short language hint used to match the user's current prompt language.
	ResponseLanguage string `json:"response_language" example:"zh"`
	// Limit caps returned artifacts.
	Limit int `json:"limit" example:"20"`
}

// ChatSessionReset is the JSON body accepted by POST /chat/session/reset.
type ChatSessionReset struct {
	// WorkspaceID is the selected workspace path or stored workspace identifier.
	WorkspaceID string `json:"workspace_id" example:"/home/user/myproject"`
	// WorkspacePath is an optional explicit workspace path; it takes precedence over WorkspaceID.
	WorkspacePath string `json:"workspace_path" example:"/home/user/myproject"`
}
