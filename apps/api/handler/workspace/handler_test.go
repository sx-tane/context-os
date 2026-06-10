package workspace

// White-box tests inspect handler internals to verify option wiring after the file split.

import "testing"

// TestNewHandlerOptionWiring verifies fluent handler options retain local cleanup directories.
func TestNewHandlerOptionWiring(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, nil, nil, nil, nil).
		WithLocalArtifactDirs("parsed", "snapshots").
		WithCodexChatSessionDir("sessions")

	if handler.parsedDir != "parsed" {
		t.Fatalf("parsedDir = %q, want parsed", handler.parsedDir)
	}
	if handler.snapshotDir != "snapshots" {
		t.Fatalf("snapshotDir = %q, want snapshots", handler.snapshotDir)
	}
	if handler.sessionDir != "sessions" {
		t.Fatalf("sessionDir = %q, want sessions", handler.sessionDir)
	}
}
