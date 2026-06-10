package presentation

// White-box tests cover trace and metadata helpers split out of presentation.go.

import "testing"

// TestBuildTraceIDIsStable verifies response trace IDs are content-addressed and repeatable.
func TestBuildTraceIDIsStable(t *testing.T) {
	t.Parallel()

	first := buildTraceID("GitHub", "owner/repo", []string{"m1", "m2"})
	second := buildTraceID("github", "owner/repo", []string{"m1", "m2"})
	if first != second {
		t.Fatalf("buildTraceID() = %q and %q, want stable IDs", first, second)
	}
}

// TestCloneMetadataDropsBlankValues verifies metadata cloning trims values and removes blank entries.
func TestCloneMetadataDropsBlankValues(t *testing.T) {
	t.Parallel()

	got := cloneMetadata(map[string]string{"keep": " value ", "drop": " "})
	if got["keep"] != "value" {
		t.Fatalf("keep metadata = %q, want value", got["keep"])
	}
	if _, ok := got["drop"]; ok {
		t.Fatal("drop metadata present, want omitted")
	}
}
