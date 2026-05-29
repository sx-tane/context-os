// package shared — white-box: tests exercise unexported helpers Preview, CloneStringMap, SetMetadata, and CapabilityStrings.
package shared

import (
	"strings"
	"testing"

	"context-os/domain/contracts"
)

// TestPreviewShort verifies Preview returns the input unchanged when it is shorter than the truncation threshold.
func TestPreviewShort(t *testing.T) {
	input := "hello world"
	got := Preview(input)
	if got != input {
		t.Fatalf("Preview() = %q, want %q", got, input)
	}
}

// TestPreviewTruncates verifies Preview truncates strings longer than 500 runes to 500 runes plus an ellipsis.
func TestPreviewTruncates(t *testing.T) {
	input := strings.Repeat("a", 600)
	got := Preview(input)
	runes := []rune(got)
	if len(runes) != 501 { // 500 + "…"
		t.Fatalf("rune count = %d, want 501", len(runes))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("Preview() = %q, want suffix '…'", got)
	}
}

// TestPreviewUnicodeSafe verifies Preview truncates at a rune boundary without producing replacement characters.
func TestPreviewUnicodeSafe(t *testing.T) {
	input := strings.Repeat("€", 600)
	got := Preview(input)
	if strings.ContainsRune(got, '\uFFFD') {
		t.Fatalf("Preview() = %q, contains replacement char U+FFFD", got)
	}
}

// TestCloneStringMapSkipsEmpty verifies CloneStringMap omits keys whose values are empty or whitespace-only.
func TestCloneStringMapSkipsEmpty(t *testing.T) {
	got := CloneStringMap(map[string]string{"a": "1", "b": "", "c": "  "})
	if _, ok := got["b"]; ok {
		t.Fatalf("CloneStringMap() key %q present, want absent", "b")
	}
	if _, ok := got["c"]; ok {
		t.Fatalf("CloneStringMap() key %q present, want absent", "c")
	}
	if got["a"] != "1" {
		t.Fatalf("CloneStringMap()[%q] = %q, want %q", "a", got["a"], "1")
	}
}

// TestSetMetadataSkipsEmpty verifies SetMetadata does not write keys with empty or whitespace-only values.
func TestSetMetadataSkipsEmpty(t *testing.T) {
	m := map[string]string{}
	SetMetadata(m, "key", "")
	SetMetadata(m, "key2", "   ")
	if len(m) != 0 {
		t.Fatalf("SetMetadata() map = %v, want empty", m)
	}
}

// TestSetMetadataSetsNonEmpty verifies SetMetadata trims and stores non-empty values under the given key.
func TestSetMetadataSetsNonEmpty(t *testing.T) {
	m := map[string]string{}
	SetMetadata(m, "key", "  value  ")
	if m["key"] != "value" {
		t.Fatalf("SetMetadata()[%q] = %q, want %q", "key", m["key"], "value")
	}
}

// TestCapabilityStringsNil verifies CapabilityStrings returns a non-nil empty slice when given nil input.
func TestCapabilityStringsNil(t *testing.T) {
	got := CapabilityStrings(nil)
	if got == nil {
		t.Fatal("CapabilityStrings(nil) = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("CapabilityStrings(nil) = %v, want empty", got)
	}
}

// TestCapabilityStringsConverts verifies CapabilityStrings converts a Capability slice to equivalent string values.
func TestCapabilityStringsConverts(t *testing.T) {
	got := CapabilityStrings([]contracts.Capability{"read", "write"})
	if len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("CapabilityStrings() = %v, want [read write]", got)
	}
}
