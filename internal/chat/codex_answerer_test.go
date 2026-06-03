package chat

import (
	"strings"
	"testing"
)

// TestLivePromptRequestsSourceSeparatedProvenance verifies live answers are prompted to preserve per-source activity detail.
func TestLivePromptRequestsSourceSeparatedProvenance(t *testing.T) {
	prompt := livePrompt("GitHub", "context-os/app#1", "what changed?")

	for _, want := range []string{
		"Structure the final answer by source",
		"exact provenance fields",
		"separate items",
		"context-os/app#1",
		"what changed?",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("livePrompt() missing %q in %q", want, prompt)
		}
	}
}
