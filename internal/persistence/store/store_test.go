package store

import (
	"reflect"
	"testing"
)

// TestWorkspaceIDFromPath verifies absolute paths are converted into stable workspace IDs.
func TestWorkspaceIDFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "nested path", path: "/Users/alice/projects/context-os", want: "Users_alice_projects_context-os"},
		{name: "root path", path: "/", want: "workspace"},
		{name: "relative path", path: "context-os", want: "context-os"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := workspaceIDFromPath(tt.path); got != tt.want {
				t.Errorf("workspaceIDFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestAliasesToPGArray verifies string slices are encoded as PostgreSQL text array literals.
func TestAliasesToPGArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want string
	}{
		{name: "empty", in: nil, want: "{}"},
		{name: "single", in: []string{"alpha"}, want: `{"alpha"}`},
		{name: "quotes", in: []string{`a"b`, "c"}, want: `{"a\"b","c"}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := aliasesToPGArray(tt.in); got != tt.want {
				t.Errorf("aliasesToPGArray(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestParsePGArray verifies PostgreSQL text array literals are decoded into Go slices.
func TestParsePGArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "{}", want: nil},
		{name: "single", raw: `{"alpha"}`, want: []string{"alpha"}},
		{name: "multiple", raw: `{"a","b","c"}`, want: []string{"a", "b", "c"}},
		{name: "escaped quote", raw: `{"a\"b"}`, want: []string{`a"b`}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parsePGArray(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePGArray(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

// TestCompactStrings verifies strings are trimmed, deduplicated, and kept in first-seen order.
func TestCompactStrings(t *testing.T) {
	t.Parallel()

	in := []string{" alpha ", "", "beta", "alpha", "beta", "gamma"}
	want := []string{"alpha", "beta", "gamma"}

	if got := compactStrings(in); !reflect.DeepEqual(got, want) {
		t.Errorf("compactStrings(%v) = %#v, want %#v", in, got, want)
	}
}

// TestWorkspaceScopedMemoryTablesIncludesUIState verifies workspace reset/delete clears durable UI state rows.
func TestWorkspaceScopedMemoryTablesIncludesUIState(t *testing.T) {
	t.Parallel()

	for _, table := range workspaceScopedMemoryTables {
		if table == "workspace_ui_state" {
			return
		}
	}
	t.Fatal("workspaceScopedMemoryTables missing workspace_ui_state")
}
