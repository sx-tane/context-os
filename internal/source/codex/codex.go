// Package codex provides a source connector that invokes Codex CLI plugins.
package codex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	// MetadataPlugin selects which installed Codex plugin should handle the request.
	MetadataPlugin = "codex_plugin"
	// MetadataPrompt preserves the exact prompt sent to Codex for replay and audit.
	MetadataPrompt = "codex_prompt"
	// MetadataCommand records the executable used for Codex CLI invocation.
	MetadataCommand = "codex_command"
	// MetadataProvider marks events produced through the Codex CLI provider path.
	MetadataProvider = "provider"
	// MetadataLog captures the combined stdout/stderr from the Codex CLI run.
	MetadataLog = "codex_log"
	// MetadataTokenOverride injects a token as the platform env var so the Codex
	// plugin authenticates as a specific account instead of the default login.
	// Value is the raw token; the connector maps it to GITHUB_TOKEN or SLACK_BOT_TOKEN
	// based on the plugin in use.
	MetadataTokenOverride = "codex_token_override"

	// PluginGitHub routes the request through the GitHub Codex plugin.
	PluginGitHub = "github"
	// PluginSlack routes the request through the Slack Codex plugin.
	PluginSlack = "slack"

	defaultCommand = "codex"
)

type connector struct {
	base      source.MCPConnector
	command   string
	workspace string
}

// NewConnector returns a source connector that shells out to Codex CLI plugins.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base:      source.NewMCPConnector("codex-cli", contracts.CapabilityRepository, contracts.CapabilityIssues, contracts.CapabilityMessages, contracts.CapabilityDocs),
		command:   defaultCommand,
		workspace: ".",
	}
}

func newConnector(command, workspace string) connector {
	return connector{
		base:      source.NewMCPConnector("codex-cli", contracts.CapabilityRepository, contracts.CapabilityIssues, contracts.CapabilityMessages, contracts.CapabilityDocs),
		command:   command,
		workspace: workspace,
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest invokes Codex CLI with the selected plugin and emits the final response as source content.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	req.Metadata = cloneMetadata(req.Metadata)
	plugin := strings.TrimSpace(req.Metadata[MetadataPlugin])
	if plugin == "" {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, errors.New("codex_plugin metadata is required"))
	}
	if plugin != PluginGitHub && plugin != PluginSlack {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, fmt.Errorf("unsupported codex plugin %q", plugin))
	}

	uri := strings.TrimSpace(req.URI)
	if uri == "" {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, errors.New("uri is required"))
	}

	prompt := promptFor(plugin, uri)

	// Allow per-request account override: inject the token as the platform env
	// var so the plugin uses a specific account rather than the Codex login session.
	var envOverrides []string
	if tok := strings.TrimSpace(req.Metadata[MetadataTokenOverride]); tok != "" {
		switch plugin {
		case PluginGitHub:
			envOverrides = append(envOverrides, "GITHUB_TOKEN="+tok)
		case PluginSlack:
			envOverrides = append(envOverrides, "SLACK_BOT_TOKEN="+tok)
		}
	}

	content, log, err := c.runCodex(ctx, prompt, envOverrides)
	if err != nil {
		return nil, err
	}

	req.Content = content
	req.Metadata[MetadataProvider] = "codex_cli"
	req.Metadata[MetadataPrompt] = prompt
	req.Metadata[MetadataCommand] = c.command
	req.Metadata[MetadataLog] = log
	req.Metadata[contracts.MetadataObjectType] = plugin
	req.Metadata[contracts.MetadataObjectID] = uri
	req.Metadata[events.MetadataSourceID] = "codex:" + plugin + ":" + uri

	return c.base.Ingest(ctx, req)
}

// runCodex invokes codex exec non-interactively and returns (content, log, error).
// It kills the entire process group on context cancellation so child processes
// spawned by the Codex agent cannot keep the stdout/stderr pipes open.
// envOverrides are KEY=VALUE pairs appended after os.Environ() so callers can
// inject account-specific tokens without modifying the global environment.
func (c connector) runCodex(ctx context.Context, prompt string, envOverrides []string) (string, string, error) {
	command := strings.TrimSpace(c.command)
	if command == "" {
		command = defaultCommand
	}

	out, err := os.CreateTemp("", "contextos-codex-*.txt")
	if err != nil {
		return "", "", c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindTemporary, true, err)
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", "", c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindTemporary, true, err)
	}
	defer func() { _ = os.Remove(outPath) }()

	args := []string{"exec", "--sandbox", "read-only", "--ephemeral", "--color", "never", "-o", outPath}
	if workspace := strings.TrimSpace(c.workspace); workspace != "" {
		args = append(args, "--cd", workspace)
	}
	args = append(args, prompt)

	// Do NOT use exec.CommandContext — it only kills the parent process.
	// We set Setpgid so the whole process group can be killed on context cancel.
	cmd := exec.Command(command, args...) //nolint:gosec
	cmd.Env = append(os.Environ(), envOverrides...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return "", "", c.commandError(err, "")
	}

	// Wait in a goroutine so we can react to context cancellation.
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	var waitErr error
	select {
	case waitErr = <-doneCh:
		// process finished normally
	case <-ctx.Done():
		// kill the entire process group (-pgid sends signal to group)
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh // drain the goroutine
		return "", "", c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindCanceled, true,
			fmt.Errorf("codex exec timed out: %w", ctx.Err()))
	}

	combinedLog := strings.TrimSpace(stdoutBuf.String() + stderrBuf.String())

	if waitErr != nil {
		return "", combinedLog, c.commandError(waitErr, combinedLog)
	}

	contentBytes, readErr := os.ReadFile(filepath.Clean(outPath))
	if readErr != nil {
		return "", combinedLog, c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindTemporary, true, readErr)
	}
	text := strings.TrimSpace(string(contentBytes))
	if text == "" {
		text = strings.TrimSpace(stdoutBuf.String())
	}
	if text == "" {
		return "", combinedLog, c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindTemporary, true,
			errors.New("codex returned no content"))
	}

	return text, combinedLog, nil
}

func (c connector) commandError(err error, output string) error {
	trimmed := strings.TrimSpace(output)
	if errors.Is(err, exec.ErrNotFound) {
		return c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindPermanent, false, errors.New("codex cli not found; install @openai/codex and try again"))
	}
	if strings.Contains(strings.ToLower(trimmed), "not logged in") || strings.Contains(strings.ToLower(trimmed), "no codex credentials") {
		return c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindPermanent, false, errors.New("codex cli is not logged in; run codex login and retry"))
	}
	if trimmed == "" {
		trimmed = err.Error()
	}
	return c.connectorError(contracts.SourceRequest{}, contracts.ErrorKindTemporary, true, fmt.Errorf("codex exec failed: %s", trimmed))
}

func promptFor(plugin, uri string) string {
	switch plugin {
	case PluginSlack:
		return "Use the Slack Codex plugin to read the Slack context identified by " + uri + ". Return the relevant channel or message content with source identifiers, timestamps, participants, and links when available. Do not modify Slack."
	case PluginGitHub:
		return "Use the GitHub Codex plugin to read the GitHub artifact identified by " + uri + ". Return the relevant repository, issue, pull request, or commit content with source identifiers, timestamps, authors, and links when available. Do not modify GitHub."
	default:
		return "Read source context for " + uri + " using the installed Codex plugin."
	}
}

func (c connector) connectorError(req contracts.SourceRequest, kind contracts.ErrorKind, retryable bool, err error) error {
	return &contracts.ConnectorError{
		Connector:  c.base.Name(),
		URI:        req.URI,
		ObjectType: req.Metadata[contracts.MetadataObjectType],
		ObjectID:   req.Metadata[contracts.MetadataObjectID],
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
