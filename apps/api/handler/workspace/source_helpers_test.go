package workspace

// White-box tests cover connected-source helper behavior split out of workspace.go.

import "testing"

// TestWorkspaceNameFromPathFallsBackForRoot verifies path-derived workspace names remain non-empty.
func TestWorkspaceNameFromPathFallsBackForRoot(t *testing.T) {
	t.Parallel()

	if got := workspaceNameFromPath("/"); got != "workspace" {
		t.Fatalf("workspaceNameFromPath() = %q, want workspace", got)
	}
}

// TestWorkspaceNameFromPathUsesLastPathSegment verifies workspace names come from the final path segment.
func TestWorkspaceNameFromPathUsesLastPathSegment(t *testing.T) {
	t.Parallel()

	if got := workspaceNameFromPath("/repo/context-os/"); got != "context-os" {
		t.Fatalf("workspaceNameFromPath() = %q, want context-os", got)
	}
}
