package chat

// White-box tests cover small shared chat helpers split out of chat.go.

import "testing"

// TestClampLimitAppliesDefaultsAndMaximum verifies local artifact query limits stay within the supported range.
func TestClampLimitAppliesDefaultsAndMaximum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: defaultLimit},
		{name: "keeps requested", limit: 5, want: 5},
		{name: "caps maximum", limit: 500, want: maxLimit},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := clampLimit(tt.limit); got != tt.want {
				t.Errorf("clampLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

// TestFirstNonEmptyTrimsValues verifies the first non-blank value is returned trimmed.
func TestFirstNonEmptyTrimsValues(t *testing.T) {
	t.Parallel()

	if got := firstNonEmpty(" ", " value ", "later"); got != "value" {
		t.Fatalf("firstNonEmpty() = %q, want value", got)
	}
}
