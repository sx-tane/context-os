// Package filesystem provides HTTP handlers for the /filesystem/* routes.
package filesystem

import (
	"encoding/json"
	"net/http"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	filesystemsource "context-os/internal/source/filesystem"
)

// Ingest handles POST /filesystem/ingest by ingesting local files
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a local filesystem artifact
// @Description  Reads a local file or folder path and returns provenance-rich events with extracted text and file metadata.
// @Tags         filesystem
// @Accept       json
// @Produce      json
// @Param        body  body      request.FilesystemIngest  true  "Filesystem ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /filesystem/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	shared.RunSourceIngest(w, r, filesystemsource.NewConnector(), decodeIngest)
}

func decodeIngest(dec *json.Decoder) (shared.SourceIngestInput, error) {
	var req request.FilesystemIngest
	if err := dec.Decode(&req); err != nil {
		return shared.SourceIngestInput{}, err
	}
	metadata := shared.CloneStringMap(req.Metadata)
	shared.SetMetadata(metadata, "filesystem_include", req.Include)
	shared.SetMetadata(metadata, "filesystem_exclude", req.Exclude)
	return shared.SourceIngestInput{
		WorkspaceID: req.WorkspaceID,
		Connector:   "filesystem",
		URI:         req.URI,
		Content:     req.Content,
		Cursor:      req.Cursor,
		Metadata:    metadata,
	}, nil
}
