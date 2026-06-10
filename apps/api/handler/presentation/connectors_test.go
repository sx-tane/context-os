package presentation

// White-box tests cover connector resolution helpers split out of presentation.go.

import (
	"testing"

	"context-os/apps/api/request"
	codexsource "context-os/internal/source/codex"
)

// TestBroadCodexSourceReturnsExamples verifies connector-only Codex requests are rejected with concrete examples.
func TestBroadCodexSourceReturnsExamples(t *testing.T) {
	t.Parallel()

	broad, examples := broadCodexSource(request.PresentationFindings{Provider: "codex", Connector: "github", URI: "github"})
	if !broad {
		t.Fatal("broadCodexSource() broad = false, want true")
	}
	if len(examples) == 0 {
		t.Fatal("examples count = 0, want concrete examples")
	}
}

// TestResolveConnectorCodexSetsPluginMetadata verifies Codex connector resolution records the selected plugin.
func TestResolveConnectorCodexSetsPluginMetadata(t *testing.T) {
	t.Parallel()

	metadata := map[string]string{}
	connector, err := resolveConnector(request.PresentationFindings{Provider: "codex", Connector: "jira", Token: "token"}, metadata)
	if err != nil {
		t.Fatalf("resolveConnector() error = %v", err)
	}
	if connector == nil {
		t.Fatal("connector = nil, want Codex connector")
	}
	if got := metadata[codexsource.MetadataPlugin]; got != codexsource.PluginAtlassianRovo {
		t.Fatalf("metadata plugin = %q, want %q", got, codexsource.PluginAtlassianRovo)
	}
	if got := metadata[codexsource.MetadataTokenOverride]; got != "token" {
		t.Fatalf("metadata token = %q, want token", got)
	}
}
