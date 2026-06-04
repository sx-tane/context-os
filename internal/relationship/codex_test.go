package relationship

// White-box package: this test injects the unexported Codex assistant constructor.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
)

// fakeCodexCommand writes a shell script to a temp dir and returns its path.
func fakeCodexCommand(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex-fake")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatalf("write fake command: %v", err)
	}
	return path
}

// TestCodexAssistantParsesCLIOutput verifies the Codex CLI assistant reads the output file relationship line.
func TestCodexAssistantParsesCLIOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf '%s' 'CONTEXTOS_RELATIONSHIPS_JSON: {"relationships":[{"from":"checkoutFeeAmount","to":"checkout_fee_amount","kind":"api_backed_by_db","evidence":"checkoutFeeAmount is stored in checkout_fee_amount","confidence":0.91}]}' > "$1"
  fi
  shift
done
`
	assistant := newCodexAssistant(fakeCodexCommand(t, script), t.TempDir())
	doc := types.NormalizedDocument{
		ID:   "d",
		Body: "checkoutFeeAmount is stored in checkout_fee_amount",
	}
	canonical := []entities.CanonicalEntity{
		{Entity: types.Entity{ID: "d:api", SourceID: "d", Type: types.APIField, Name: "checkoutFeeAmount"}},
		{Entity: types.Entity{ID: "d:db", SourceID: "d", Type: types.DBColumn, Name: "checkout_fee_amount"}},
	}

	got, err := assistant.ProposeRelationships(context.Background(), doc, canonical)
	if err != nil {
		t.Fatalf("ProposeRelationships() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ProposeRelationships() length = %d, want 1", len(got))
	}
	if got[0].Metadata[MetadataAssistProvider] != AssistProviderCodexCLI {
		t.Fatalf("Metadata[assist_provider] = %q, want codex_cli", got[0].Metadata[MetadataAssistProvider])
	}
}
