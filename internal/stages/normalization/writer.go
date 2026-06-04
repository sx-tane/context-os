package normalization

import (
	"encoding/json" // serialises NormalizedDocument to JSON
	"fmt"           // formats error messages
	"os"            // creates directories and writes files
	"path/filepath" // builds the workspace sub-directory and file path

	"context-os/domain/types" // NormalizedDocument type produced by Normalize
)

// DocumentWriter persists NormalizedDocument values to disk under a configurable
// root directory. The on-disk layout is:
//
//	<dir>/<workspaceID>/<docID>.json
//
// This lets downstream stages replay or diff parsed output without re-ingesting
// the original source. The zero value is not usable; construct with NewDocumentWriter.
type DocumentWriter struct {
	dir string // root directory under which workspace subdirectories are created
}

// NewDocumentWriter returns a DocumentWriter that stores files under dir.
// Pass an empty dir to create a no-op writer (Write becomes a no-op).
func NewDocumentWriter(dir string) *DocumentWriter {
	return &DocumentWriter{dir: dir}
}

// Write serialises doc to <dir>/<workspaceID>/<doc.ID>.json. It creates the
// workspace subdirectory when needed. An empty dir or doc.ID is a no-op.
func (w *DocumentWriter) Write(workspaceID string, doc types.NormalizedDocument) error {
	if w.dir == "" || doc.ID == "" {
		return nil
	}
	wsDir := filepath.Join(w.dir, workspaceID)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return fmt.Errorf("normalization: create parsed dir: %w", err)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("normalization: marshal document %s: %w", doc.ID, err)
	}
	dest := filepath.Join(wsDir, doc.ID+".json")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("normalization: write document %s: %w", doc.ID, err)
	}
	return nil
}
