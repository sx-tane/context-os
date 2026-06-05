package graphverify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"context-os/domain/types"
	"context-os/internal/codexio"
)

const (
	defaultCodexCommand = "codex"
	outputPrefix        = "CONTEXTOS_GRAPH_VERIFY_JSON:"
	maxSnapshotChars    = 20000
)

// CodexAssistant invokes Codex CLI to verify cross-source graph relationships.
type CodexAssistant struct {
	command   string
	workspace string
}

// NewCodexAssistant returns a Codex-backed graph verifier.
func NewCodexAssistant() *CodexAssistant {
	return &CodexAssistant{command: resolveCodexCommand(), workspace: "."}
}

// Provider returns the verifier provenance label.
func (a *CodexAssistant) Provider() string {
	return "codex_cli"
}

// Verify asks Codex for relationship proposals and parses the strict JSON response.
func (a *CodexAssistant) Verify(ctx context.Context, snapshot Snapshot) ([]types.Relationship, error) {
	if a == nil {
		return nil, nil
	}
	output, err := a.run(ctx, graphPrompt(snapshot))
	if err != nil {
		return nil, err
	}
	return ParseOutput(output)
}

func (a *CodexAssistant) run(ctx context.Context, prompt string) (string, error) {
	command := strings.TrimSpace(a.command)
	if command == "" {
		command = defaultCodexCommand
	}
	out, err := os.CreateTemp("", "contextos-graph-verify-*.txt")
	if err != nil {
		return "", fmt.Errorf("create graph verifier output: %w", err)
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close graph verifier output: %w", err)
	}
	defer func() { _ = os.Remove(outPath) }()

	args := []string{"exec", "--sandbox", "read-only", "--ephemeral", "--color", "never", "-o", outPath}
	if workspace := strings.TrimSpace(a.workspace); workspace != "" {
		args = append(args, "--cd", workspace)
	}
	args = append(args, prompt)

	cmd := exec.Command(command, args...) //nolint:gosec
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutBuf := codexio.NewBoundedBuffer(codexio.DefaultLogLimit)
	stderrBuf := codexio.NewBoundedBuffer(codexio.DefaultLogLimit)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	if err := cmd.Start(); err != nil {
		return "", codexCommandError(err, "")
	}
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	var waitErr error
	select {
	case waitErr = <-doneCh:
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-doneCh
		return "", fmt.Errorf("graph verifier timed out: %w", ctx.Err())
	}

	combined := strings.TrimSpace(stdoutBuf.String() + stderrBuf.String())
	if waitErr != nil {
		return "", codexCommandError(waitErr, combined)
	}
	contentBytes, err := os.ReadFile(filepath.Clean(outPath))
	if err != nil {
		return "", fmt.Errorf("read graph verifier output: %w", err)
	}
	output := strings.TrimSpace(string(contentBytes))
	if output == "" {
		output = strings.TrimSpace(stdoutBuf.String())
	}
	if output == "" {
		return "", errors.New("graph verifier returned no output")
	}
	return output, nil
}

// ParseOutput parses the strict graph verifier JSON line.
func ParseOutput(output string) ([]types.Relationship, error) {
	line, ok := jsonLine(output)
	if !ok {
		return nil, nil
	}
	var envelope struct {
		Relationships []struct {
			From       string   `json:"from"`
			To         string   `json:"to"`
			Kind       string   `json:"kind"`
			Evidence   []string `json:"evidence_ids"`
			Confidence float64  `json:"confidence"`
		} `json:"relationships"`
	}
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		return nil, fmt.Errorf("parse graph verifier output: %w", err)
	}
	out := make([]types.Relationship, 0, len(envelope.Relationships))
	for _, item := range envelope.Relationships {
		out = append(out, types.Relationship{
			FromID:     strings.TrimSpace(item.From),
			ToID:       strings.TrimSpace(item.To),
			Kind:       types.RelationshipKind(strings.TrimSpace(item.Kind)),
			Confidence: item.Confidence,
			Evidence:   item.Evidence,
		})
	}
	return out, nil
}

func jsonLine(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, outputPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, outputPrefix)), true
		}
	}
	return "", false
}

func graphPrompt(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("You are verifying ContextOS cross-source graph relationships from a local evidence snapshot.\n")
	b.WriteString("Compare rows, evidence IDs, and source provenance before proposing edges.\n")
	b.WriteString("Return exactly one compact line beginning with ")
	b.WriteString(outputPrefix)
	b.WriteString(" followed by JSON shaped as ")
	b.WriteString(`{"relationships":[{"from":"entity-id","to":"entity-id","kind":"requirement_affects_service","evidence_ids":["event-id"],"confidence":0.86}]}`)
	b.WriteString(". Use only entity IDs and evidence IDs in the snapshot. Do not invent entities or evidence. Require cross-source evidence. Allowed kinds: ")
	b.WriteString(strings.Join(relationshipKindNames(), ", "))
	b.WriteString(". If no relationship is clearly supported, return ")
	b.WriteString(outputPrefix)
	b.WriteString(` {"relationships":[]}`)
	b.WriteString(".\n\n")
	b.WriteString(limitText(snapshotText(snapshot), maxSnapshotChars))
	return b.String()
}

func snapshotText(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("Workspace: ")
	b.WriteString(snapshot.WorkspaceID)
	b.WriteString("\nTrace: ")
	b.WriteString(snapshot.TraceID)
	b.WriteString("\n\nEntities:\n")
	entities := append([]string(nil), entityLines(snapshot)...)
	sort.Strings(entities)
	for _, line := range entities {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\nEvidence:\n")
	for _, event := range snapshot.Events {
		b.WriteString("- id: ")
		b.WriteString(event.ID)
		b.WriteString("; connector: ")
		b.WriteString(event.Connector)
		b.WriteString("; source: ")
		b.WriteString(event.SourceURI)
		b.WriteString("; title: ")
		b.WriteString(event.Title)
		b.WriteString("; body: ")
		b.WriteString(strings.Join(strings.Fields(event.Body), " "))
		b.WriteString("\n")
	}
	return b.String()
}

func entityLines(snapshot Snapshot) []string {
	out := make([]string, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		out = append(out, fmt.Sprintf("- id: %s; name: %s; type: %s; source_id: %s", entity.Entity.ID, entity.Entity.Name, entity.Entity.Type, entity.Entity.SourceID))
	}
	return out
}

func relationshipKindNames() []string {
	return []string{
		string(types.CoOccursInDocument),
		string(types.RequirementAffectsAPI),
		string(types.RequirementAffectsService),
		string(types.APIBackedByDB),
		string(types.EnumConstrainsField),
		string(types.ServiceDependsOn),
	}
}

func limitText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

func codexCommandError(err error, output string) error {
	trimmed := strings.TrimSpace(output)
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("codex cli not found; install @openai/codex and try again")
	}
	if strings.Contains(strings.ToLower(trimmed), "not logged in") || strings.Contains(strings.ToLower(trimmed), "no codex credentials") {
		return errors.New("codex cli is not logged in; run codex login and retry")
	}
	if trimmed == "" {
		trimmed = err.Error()
	}
	return fmt.Errorf("graph verifier failed: %s", trimmed)
}

func resolveCodexCommand() string {
	if configured := strings.TrimSpace(os.Getenv("CODEX_BIN")); configured != "" {
		return configured
	}
	if path, err := exec.LookPath(defaultCodexCommand); err == nil {
		return path
	}
	return defaultCodexCommand
}
