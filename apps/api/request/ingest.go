// Package request defines inbound JSON request types for the ContextOS API.
package request

// GithubIngest is the JSON body accepted by POST /github/ingest.
type GithubIngest struct {
	URI      string `json:"uri"      example:"https://github.com/sx-tane/context-os/issues/1"`
	Token    string `json:"token"    example:"ghp_xxxx"`
	Provider string `json:"provider" example:"token"`
}

// SlackIngest is the JSON body accepted by POST /slack/ingest.
type SlackIngest struct {
	URI      string `json:"uri"      example:"slack://C1234567890"`
	Token    string `json:"token"    example:"xoxb-111-222-xxxx"`
	Provider string `json:"provider" example:"token"`
}

// JiraIngest is the JSON body accepted by POST /jira/ingest.
type JiraIngest struct {
	URI        string            `json:"uri"          example:"https://mysite.atlassian.net/browse/PROJ-123"`
	Content    string            `json:"content"      example:"Optional raw content override"`
	Cursor     string            `json:"cursor"       example:"eyJsYXN0X2lkIjoiMTIzIn0="`
	Token      string            `json:"token"        example:"ATATT3xFfGF0xxxx"`
	Provider   string            `json:"provider"     example:"token"`
	Email      string            `json:"email"        example:"user@example.com"`
	APIBaseURL string            `json:"api_base_url" example:"https://mysite.atlassian.net"`
	Expand     string            `json:"expand"       example:"renderedFields,names"`
	Metadata   map[string]string `json:"metadata"`
}

// GoogleDriveIngest is the JSON body accepted by POST /googledrive/ingest.
type GoogleDriveIngest struct {
	URI                string            `json:"uri"                  example:"https://drive.google.com/drive/folders/1234567890"`
	FolderID           string            `json:"folder_id"            example:"1234567890"`
	CredentialPath     string            `json:"credential_path"      example:"/Users/name/.config/context-os/google-authorized-user.json"`
	ServiceAccountPath string            `json:"service_account_path" example:"/Users/name/.config/context-os/google-service-account.json"`
	AccessToken        string            `json:"access_token"         example:"ya29.a0AfH6SMD..."`
	Cursor             string            `json:"cursor"               example:"2026-05-29T10:00:00Z"`
	Provider           string            `json:"provider"             example:"token"`
	Metadata           map[string]string `json:"metadata"`
}

// FilesystemIngest is the JSON body accepted by POST /filesystem/ingest for local files or folders.
type FilesystemIngest struct {
	URI      string            `json:"uri"     example:"storage/raw/README.md"`
	Content  string            `json:"content" example:"Optional raw content instead of reading from URI"`
	Cursor   string            `json:"cursor"  example:"eyJsYXN0X2lkIjoiMTIzIn0="`
	Include  string            `json:"include" example:"*.go,*.md"`
	Exclude  string            `json:"exclude" example:"node_modules,*.log"`
	Metadata map[string]string `json:"metadata"`
}
