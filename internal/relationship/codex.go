package relationship

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"context-os/domain/entities"
	"context-os/domain/types"
)

const (
	defaultCodexCommand = "codex"
	maxPromptBodyChars  = 12000
)

// CodexAssistant invokes the local Codex CLI to propose same-document relationships.
type CodexAssistant struct {
	command   string
	workspace string
}

// NewCodexAssistant returns a relationship assistant backed by the local Codex CLI.
func NewCodexAssistant() *CodexAssistant {
	return &CodexAssistant{command: resolveCodexCommand(), workspace: "."}
}

func newCodexAssistant(command, workspace string) *CodexAssistant {
	return &CodexAssistant{command: command, workspace: workspace}
}

// Provider returns the provenance label for accepted Codex CLI relationship proposals.
func (a *CodexAssistant) Provider() string {
	return AssistProviderCodexCLI
}

// ProposeRelationships asks Codex for relationship proposals and parses accepted edges.
func (a *CodexAssistant) ProposeRelationships(ctx context.Context, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) ([]types.Relationship, error) {
	if a == nil {
		return nil, nil
	}
	output, err := a.run(ctx, relationshipPrompt(doc, canonical))
	if err != nil {
		return nil, err
	}
	return ParseAssistantOutput(output, doc, canonical)
}

func (a *CodexAssistant) run(ctx context.Context, prompt string) (string, error) {
	command := strings.TrimSpace(a.command)
	if command == "" {
		command = defaultCodexCommand
	}

	out, err := os.CreateTemp("", "contextos-relationships-*.txt")
	if err != nil {
		return "", fmt.Errorf("create codex relationship output: %w", err)
	}
	outPath := out.Name()
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close codex relationship output: %w", err)
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

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

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
		return "", fmt.Errorf("codex relationship assist timed out: %w", ctx.Err())
	}

	combined := strings.TrimSpace(stdoutBuf.String() + stderrBuf.String())
	if waitErr != nil {
		return "", codexCommandError(waitErr, combined)
	}

	contentBytes, err := os.ReadFile(filepath.Clean(outPath))
	if err != nil {
		return "", fmt.Errorf("read codex relationship output: %w", err)
	}
	output := strings.TrimSpace(string(contentBytes))
	if output == "" {
		output = strings.TrimSpace(stdoutBuf.String())
	}
	if output == "" {
		return "", errors.New("codex relationship assist returned no output")
	}
	return output, nil
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
	return fmt.Errorf("codex relationship assist failed: %s", trimmed)
}

func relationshipPrompt(doc types.NormalizedDocument, canonical []entities.CanonicalEntity) string {
	entities := append([]entities.CanonicalEntity(nil), canonical...)
	sort.SliceStable(entities, func(i, j int) bool {
		if entities[i].Entity.Name == entities[j].Entity.Name {
			return entities[i].Entity.ID < entities[j].Entity.ID
		}
		return entities[i].Entity.Name < entities[j].Entity.Name
	})

	var b strings.Builder
	b.WriteString("You are assisting ContextOS relationship extraction for one source document.\n")
	b.WriteString("Return exactly one compact line beginning with ")
	b.WriteString(AssistOutputPrefix)
	b.WriteString(" followed by JSON shaped as ")
	b.WriteString(`{"relationships":[{"from":"entity name","to":"entity name","kind":"api_backed_by_db","evidence":"exact source quote","confidence":0.86}]}`)
	b.WriteString(". Use only existing entity names from the list. Do not invent entities, do not cross documents, and do not delete baseline relationships. ")
	b.WriteString("Evidence must be an exact quote from the title or body. Use confidence >= 0.75 only when the quote directly supports the edge. ")
	b.WriteString("Allowed relationship kinds: ")
	b.WriteString(strings.Join(relationshipKindNames(), ", "))
	b.WriteString(". If no supported relationship is explicitly evidenced, return ")
	b.WriteString(AssistOutputPrefix)
	b.WriteString(` {"relationships":[]}`)
	b.WriteString(".\n\n")
	b.WriteString("Document ID: ")
	b.WriteString(doc.ID)
	b.WriteString("\nTitle:\n")
	b.WriteString(limitText(doc.Title, maxPromptBodyChars/4))
	b.WriteString("\n\nBody:\n")
	b.WriteString(limitText(doc.Body, maxPromptBodyChars))
	b.WriteString("\n\nEntities:\n")
	for _, canonicalEntity := range entities {
		entity := canonicalEntity.Entity
		b.WriteString("- name: ")
		b.WriteString(entity.Name)
		b.WriteString("; id: ")
		b.WriteString(entity.ID)
		b.WriteString("; type: ")
		b.WriteString(string(entity.Type))
		if len(entity.Aliases) > 0 {
			b.WriteString("; aliases: ")
			b.WriteString(strings.Join(entity.Aliases, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
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

func resolveCodexCommand() string {
	if configured := strings.TrimSpace(os.Getenv("CODEX_BIN")); configured != "" {
		return configured
	}
	if path, err := exec.LookPath(defaultCodexCommand); err == nil {
		return path
	}
	return defaultCodexCommand
}
