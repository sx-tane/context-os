package workspace

// White-box tests cover package-local helper behavior split out of workspace.go.

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestReadBoundedBodyRejectsInvalidJSON verifies bounded body reads fail before invalid payloads reach handlers.
func TestReadBoundedBodyRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("PUT", "/workspace/analysis-basket", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	if _, err := readBoundedBody(rec, req, 64); err == nil {
		t.Fatal("readBoundedBody() error = nil, want invalid JSON error")
	}
}

// TestWorkspaceRefFromQueryPrefersWorkspaceID verifies workspace_id takes precedence over workspace_path query scope.
func TestWorkspaceRefFromQueryPrefersWorkspaceID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/workspace/analysis-basket?workspace_id=ws-id&workspace_path=/workspace", nil)
	if got := workspaceRefFromQuery(req); got != "ws-id" {
		t.Fatalf("workspaceRefFromQuery() = %q, want ws-id", got)
	}
}

// TestFirstNonEmptyPreservesOriginalSpacing verifies the selected value is returned exactly as provided.
func TestFirstNonEmptyPreservesOriginalSpacing(t *testing.T) {
	t.Parallel()

	if got := firstNonEmpty("", " value ", "later"); got != " value " {
		t.Fatalf("firstNonEmpty() = %q, want original spaced value", got)
	}
}
