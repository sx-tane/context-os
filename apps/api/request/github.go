// Package request defines inbound JSON request types for the ContextOS API.
package request

// GithubIngest is the JSON body accepted by POST /github/ingest.
type GithubIngest struct {
	URI      string `json:"uri"`
	Token    string `json:"token"`
	Provider string `json:"provider"`
}
