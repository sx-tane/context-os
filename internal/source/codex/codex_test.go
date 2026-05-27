package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"context-os/domain/contracts"
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

// fakeExec is the standard fake that writes content to -o <file> and logs to stdout.
const fakeExecScript = `#!/bin/sh
echo "fake codex header"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf 'codex plugin content' > "$1"
  fi
  shift
done
`

func TestConnectorUsesCodexExecOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	connector := newConnector(fakeCodexCommand(t, fakeExecScript), t.TempDir())

	events, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:      "slack://C1234567890",
		Metadata: map[string]string{MetadataPlugin: PluginSlack},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Ingest() returned %d events, want 1", len(events))
	}
	if events[0].Content != "codex plugin content" {
		t.Fatalf("event content = %q", events[0].Content)
	}
	if events[0].Metadata[MetadataProvider] != "codex_cli" {
		t.Fatalf("provider metadata = %q", events[0].Metadata[MetadataProvider])
	}
	if events[0].Metadata[MetadataPlugin] != PluginSlack {
		t.Fatalf("plugin metadata = %q", events[0].Metadata[MetadataPlugin])
	}
}

func TestConnectorCapturesLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	connector := newConnector(fakeCodexCommand(t, fakeExecScript), t.TempDir())

	events, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:      "https://github.com/owner/repo/issues/1",
		Metadata: map[string]string{MetadataPlugin: PluginGitHub},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	log := events[0].Metadata[MetadataLog]
	if !strings.Contains(log, "fake codex header") {
		t.Fatalf("expected codex_log to contain stdout output, got %q", log)
	}
}

func TestConnectorPassesEphemeralAndColorFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	argsFile := t.TempDir() + "/args.txt"
	script := `#!/bin/sh
printf '%s\n' "$@" > '` + argsFile + `'
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then shift; printf 'ok' > "$1"; fi
  shift
done
`
	connector := newConnector(fakeCodexCommand(t, script), t.TempDir())
	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:      "slack://C1234567890",
		Metadata: map[string]string{MetadataPlugin: PluginSlack},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	data, _ := os.ReadFile(argsFile)
	args := string(data)
	for _, flag := range []string{"--ephemeral", "--color", "never", "--sandbox", "read-only", "-o"} {
		if !strings.Contains(args, flag) {
			t.Errorf("expected arg %q in codex call, got:\n%s", flag, args)
		}
	}
}

func TestConnectorContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	// Script that sleeps longer than the context deadline.
	slowScript := `#!/bin/sh
sleep 30
`
	connector := newConnector(fakeCodexCommand(t, slowScript), t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := connector.Ingest(ctx, contracts.SourceRequest{
		URI:      "slack://C1234567890",
		Metadata: map[string]string{MetadataPlugin: PluginSlack},
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected timeout/canceled error, got: %v", err)
	}
}

func TestConnectorNotLoggedIn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	notLoggedIn := `#!/bin/sh
echo "Not logged in" >&2
exit 1
`
	connector := newConnector(fakeCodexCommand(t, notLoggedIn), t.TempDir())

	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:      "slack://C1234567890",
		Metadata: map[string]string{MetadataPlugin: PluginSlack},
	})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("expected login error, got: %v", err)
	}
}

func TestConnectorRequiresSupportedPlugin(t *testing.T) {
	connector := newConnector("unused", t.TempDir())

	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "slack://C1234567890",
	})
	if err == nil {
		t.Fatal("Ingest() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "codex_plugin metadata is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestConnectorRejectsUnknownPlugin(t *testing.T) {
	connector := newConnector("unused", t.TempDir())

	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:      "slack://C1234567890",
		Metadata: map[string]string{MetadataPlugin: "jira"},
	})
	if err == nil {
		t.Fatal("Ingest() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unsupported codex plugin`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestConnectorRequiresURI(t *testing.T) {
	connector := newConnector("unused", t.TempDir())

	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		Metadata: map[string]string{MetadataPlugin: PluginGitHub},
	})
	if err == nil {
		t.Fatal("Ingest() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "uri is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestConnectorTokenOverrideInjected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	// Script writes the value of GITHUB_TOKEN to the output file so we can
	// assert the env var was injected correctly.
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf '%s' "$GITHUB_TOKEN" > "$1"
  fi
  shift
done
`
	connector := newConnector(fakeCodexCommand(t, script), t.TempDir())

	events, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "https://github.com/owner/repo/issues/1",
		Metadata: map[string]string{
			MetadataPlugin:        PluginGitHub,
			MetadataTokenOverride: "ghp_testtoken123",
		},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if events[0].Content != "ghp_testtoken123" {
		t.Fatalf("expected GITHUB_TOKEN injected as content, got %q", events[0].Content)
	}
}

func TestConnectorSlackTokenOverrideInjected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake command is unix-only")
	}

	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf '%s' "$SLACK_BOT_TOKEN" > "$1"
  fi
  shift
done
`
	connector := newConnector(fakeCodexCommand(t, script), t.TempDir())

	events, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "slack://C1234567890",
		Metadata: map[string]string{
			MetadataPlugin:        PluginSlack,
			MetadataTokenOverride: "xoxb-testtoken456",
		},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if events[0].Content != "xoxb-testtoken456" {
		t.Fatalf("expected SLACK_BOT_TOKEN injected as content, got %q", events[0].Content)
	}
}
