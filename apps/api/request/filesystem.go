package request

// FilesystemIngest is the JSON body accepted by POST /filesystem/ingest for local files or folders.
type FilesystemIngest struct {
	URI      string            `json:"uri"`
	Content  string            `json:"content"`
	Cursor   string            `json:"cursor"`
	Include  string            `json:"include"`
	Exclude  string            `json:"exclude"`
	Metadata map[string]string `json:"metadata"`
}
