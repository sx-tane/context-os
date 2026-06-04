package normalization_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"context-os/domain/types"
	"context-os/internal/stages/normalization"
)

func sampleDoc(id string) types.NormalizedDocument {
	return types.NormalizedDocument{
		ID:           id,
		Source:       "test",
		SourceType:   "issue",
		Title:        "Test Document",
		Body:         "Body text for " + id,
		ContentHash:  "hash-" + id,
		RuleVersion:  normalization.RuleVersion,
		NormalizedAt: time.Now().UTC(),
	}
}

// TestDocumentWriterNoopOnEmptyDir verifies that Write is a no-op and returns nil when dir is empty.
func TestDocumentWriterNoopOnEmptyDir(t *testing.T) {
	w := normalization.NewDocumentWriter("")
	if err := w.Write("ws1", sampleDoc("doc1")); err != nil {
		t.Errorf("DocumentWriter.Write() with empty dir error = %v; want nil", err)
	}
}

// TestDocumentWriterNoopOnEmptyID verifies that Write is a no-op when doc.ID is empty.
func TestDocumentWriterNoopOnEmptyID(t *testing.T) {
	dir := t.TempDir()
	w := normalization.NewDocumentWriter(dir)
	if err := w.Write("ws1", sampleDoc("")); err != nil {
		t.Errorf("DocumentWriter.Write() with empty ID error = %v; want nil", err)
	}
}

// TestDocumentWriterRoundTrip verifies that a written document can be read back and matches the original.
func TestDocumentWriterRoundTrip(t *testing.T) {
	dir := t.TempDir()
	w := normalization.NewDocumentWriter(dir)
	doc := sampleDoc("event-abc")

	if err := w.Write("workspace-1", doc); err != nil {
		t.Fatalf("DocumentWriter.Write() error = %v", err)
	}

	dest := filepath.Join(dir, "workspace-1", "event-abc.json")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", dest, err)
	}

	var got types.NormalizedDocument
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.ID != doc.ID {
		t.Errorf("ID = %q; want %q", got.ID, doc.ID)
	}
	if got.Title != doc.Title {
		t.Errorf("Title = %q; want %q", got.Title, doc.Title)
	}
}

// TestDocumentWriterCreatesWorkspaceSubdir verifies that Write creates the workspace subdirectory.
func TestDocumentWriterCreatesWorkspaceSubdir(t *testing.T) {
	dir := t.TempDir()
	w := normalization.NewDocumentWriter(dir)

	if err := w.Write("ws-new", sampleDoc("doc-x")); err != nil {
		t.Fatalf("DocumentWriter.Write() error = %v", err)
	}

	wsDir := filepath.Join(dir, "ws-new")
	info, err := os.Stat(wsDir)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", wsDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", wsDir)
	}
}
