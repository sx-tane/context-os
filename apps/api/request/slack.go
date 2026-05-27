package request

// SlackIngest is the JSON body accepted by POST /slack/ingest.
type SlackIngest struct {
	URI      string `json:"uri"`
	Token    string `json:"token"`
	Provider string `json:"provider"`
}
