package presentation

// White-box tests cover small presentation helper functions split out of presentation.go.

import (
	"reflect"
	"testing"

	"context-os/domain/types"
)

// TestCollectMismatchIDsSortsIDs verifies mismatch IDs are sorted for stable response traces.
func TestCollectMismatchIDsSortsIDs(t *testing.T) {
	t.Parallel()

	got := collectMismatchIDs([]types.Mismatch{{ID: "z"}, {ID: "a"}})
	want := []string{"a", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectMismatchIDs() = %#v, want %#v", got, want)
	}
}

// TestSeverityCountDefaultsBlankSeverityToMedium verifies empty severities count as medium.
func TestSeverityCountDefaultsBlankSeverityToMedium(t *testing.T) {
	t.Parallel()

	got := severityCount([]types.Mismatch{{Severity: "high"}, {Severity: ""}})
	if got["high"] != 1 {
		t.Fatalf("high count = %d, want 1", got["high"])
	}
	if got["medium"] != 1 {
		t.Fatalf("medium count = %d, want 1", got["medium"])
	}
}
