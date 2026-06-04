// Package request defines inbound JSON request types for the ContextOS API.
package request

// GithubIngest is the JSON body accepted by POST /github/ingest.
type GithubIngest struct {
	WorkspaceID string `json:"workspace_id" example:"/home/user/myproject"`
	URI         string `json:"uri"          example:"https://github.com/owner/repo/issues/1"`
	Token       string `json:"token"        example:"ghp_xxxx"`
	Provider    string `json:"provider"     example:"token"`
}

// SlackIngest is the JSON body accepted by POST /slack/ingest.
type SlackIngest struct {
	WorkspaceID string `json:"workspace_id" example:"/home/user/myproject"`
	URI         string `json:"uri"          example:"slack://CHANNEL_ID"`
	Token       string `json:"token"        example:"xoxb-111-222-xxxx"`
	Provider    string `json:"provider"     example:"token"`
}

// JiraIngest is the JSON body accepted by POST /jira/ingest.
type JiraIngest struct {
	WorkspaceID string            `json:"workspace_id" example:"/home/user/myproject"`
	URI         string            `json:"uri"          example:"https://site.atlassian.net/browse/PROJ-123"`
	Content     string            `json:"content"      example:"Optional raw content override"`
	Cursor      string            `json:"cursor"       example:"eyJsYXN0X2lkIjoiMTIzIn0="`
	Token       string            `json:"token"        example:"ATATT3xFfGF0xxxx"`
	Provider    string            `json:"provider"     example:"token"`
	Email       string            `json:"email"        example:"user@example.com"`
	APIBaseURL  string            `json:"api_base_url" example:"https://mysite.atlassian.net"`
	Expand      string            `json:"expand"       example:"renderedFields,names"`
	Metadata    map[string]string `json:"metadata"`
}

// GoogleDriveIngest is the JSON body accepted by POST /googledrive/ingest.
type GoogleDriveIngest struct {
	WorkspaceID        string            `json:"workspace_id"          example:"/home/user/myproject"`
	URI                string            `json:"uri"                  example:"https://drive.google.com/drive/folders/1234567890"`
	FolderID           string            `json:"folder_id"            example:"1234567890"`
	CredentialPath     string            `json:"credential_path"      example:"/Users/name/.config/context-os/google-authorized-user.json"`
	ServiceAccountPath string            `json:"service_account_path" example:"/Users/name/.config/context-os/google-service-account.json"`
	AccessToken        string            `json:"access_token"         example:"ya29.a0AfH6SMD..."`
	Cursor             string            `json:"cursor"               example:"2026-05-29T10:00:00Z"`
	Provider           string            `json:"provider"             example:"token"`
	Metadata           map[string]string `json:"metadata"`
}

// NotionIngest is the JSON body accepted by POST /notion/ingest.
type NotionIngest struct {
	WorkspaceID string            `json:"workspace_id" example:"/home/user/myproject"`
	URI         string            `json:"uri"          example:"notion://page/PAGE_ID"`
	Content     string            `json:"content"      example:"Optional raw content instead of fetching from Notion"`
	Token       string            `json:"token"        example:"secret_xxxx"`
	Provider    string            `json:"provider"     example:"token"`
	Metadata    map[string]string `json:"metadata"`
}

// SharePointIngest is the JSON body accepted by POST /sharepoint/ingest.
type SharePointIngest struct {
	WorkspaceID  string            `json:"workspace_id"   example:"/home/user/myproject"`
	URI          string            `json:"uri"           example:"sharepoint://sites/mysite/items/abcdef01"`
	Content      string            `json:"content"       example:"Optional raw content instead of fetching from Graph"`
	Token        string            `json:"token"         example:"eyJ0..."`
	TenantID     string            `json:"tenant_id"     example:"00000000-0000-0000-0000-000000000000"`
	ClientID     string            `json:"client_id"     example:"11111111-1111-1111-1111-111111111111"`
	ClientSecret string            `json:"client_secret" example:"secret~value"`
	Provider     string            `json:"provider"      example:"token"`
	Metadata     map[string]string `json:"metadata"`
}

// FilesystemIngest is the JSON body accepted by POST /filesystem/ingest for local files or folders.
type FilesystemIngest struct {
	WorkspaceID string            `json:"workspace_id" example:"/home/user/myproject"`
	URI         string            `json:"uri"          example:"storage/raw/README.md"`
	Content     string            `json:"content"      example:"Optional raw content instead of reading from URI"`
	Cursor      string            `json:"cursor"       example:"eyJsYXN0X2lkIjoiMTIzIn0="`
	Include     string            `json:"include"      example:"*.go,*.md"`
	Exclude     string            `json:"exclude"      example:"node_modules,*.log"`
	Metadata    map[string]string `json:"metadata"`
}
