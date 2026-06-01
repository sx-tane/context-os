// Package request defines inbound JSON request types for presentation and findings APIs.
package request

// PresentationFindings is the JSON body accepted by POST /presentation/findings.
type PresentationFindings struct {
	// WorkspaceID is the optional workspace path used to scope and persist results.
	// When set the pipeline output is written to Postgres and cache-hit logic applies.
	WorkspaceID      string            `json:"workspace_id"      example:"/home/user/myproject"`
	Connector        string            `json:"connector"         example:"filesystem"`
	URI              string            `json:"uri"               example:"storage/raw/context.txt"`
	Content          string            `json:"content"           example:"frontend expects refundStatus but backend exposes missingRefundState"`
	Cursor           string            `json:"cursor"            example:"eyJsYXN0X2lkIjoiMTIzIn0="`
	Provider         string            `json:"provider"          example:"token"`
	Token            string            `json:"token"             example:"ghp_xxxx"`
	Role             string            `json:"role"              example:"pmo"`
	IncludeExecution *bool             `json:"include_execution"`
	// ForceRefresh bypasses the findings cache and always re-ingests from source.
	ForceRefresh     bool              `json:"force_refresh"`
	Metadata         map[string]string `json:"metadata"`
}