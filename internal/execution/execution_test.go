package execution_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"context-os/internal/execution"
)

// TestLocalStubExecutorAnalyzeReturnsDeterministicResult verifies the local stub returns a stable summary and prompt metadata.
func TestLocalStubExecutorAnalyzeReturnsDeterministicResult(t *testing.T) {
	got, err := (execution.LocalStubExecutor{}).Analyze(context.Background(), execution.CodexRequest{Prompt: "summarize drift"})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if got.Summary == "" {
		t.Fatalf("Summary = %q, want local stub summary", got.Summary)
	}
	if got.Metadata["mode"] != "local-stub" {
		t.Fatalf("Metadata[mode] = %q, want local-stub", got.Metadata["mode"])
	}
	if got.Metadata["prompt"] != "summarize drift" {
		t.Fatalf("Metadata[prompt] = %q, want summarize drift", got.Metadata["prompt"])
	}
}

// TestLocalStubExecutorAnalyzeRespectsCancellation verifies the local stub returns context cancellation before producing output.
func TestLocalStubExecutorAnalyzeRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (execution.LocalStubExecutor{}).Analyze(ctx, execution.CodexRequest{Prompt: "summarize drift"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Analyze() error = %v, want %v", err, context.Canceled)
	}
}

// TestTemplateExecutorFallsBackToStubWhenFileIsMissing verifies that TemplateExecutor returns a stub result when the template file does not exist.
func TestTemplateExecutorFallsBackToStubWhenFileIsMissing(t *testing.T) {
	exec := execution.TemplateExecutor{PromptsDir: t.TempDir()}
	got, err := exec.Analyze(context.Background(), execution.CodexRequest{Prompt: "nonexistent"})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if got.Metadata["mode"] != "template-stub" {
		t.Errorf("Metadata[mode] = %q; want template-stub", got.Metadata["mode"])
	}
}

// TestTemplateExecutorRendersTemplateWithVariables verifies that TemplateExecutor loads a template file and substitutes context variables.
func TestTemplateExecutorRendersTemplateWithVariables(t *testing.T) {
	dir := t.TempDir()
	content := "Hello {{name}}, workspace is {{workspace}}."
	if err := os.WriteFile(filepath.Join(dir, "greet.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	exec := execution.TemplateExecutor{PromptsDir: dir}
	got, err := exec.Analyze(context.Background(), execution.CodexRequest{
		Prompt:  "greet",
		Context: map[string]string{"name": "Alice", "workspace": "/proj"},
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if got.Metadata["mode"] != "template" {
		t.Errorf("Metadata[mode] = %q; want template", got.Metadata["mode"])
	}
	want := "Hello Alice, workspace is /proj."
	if got.Summary != want {
		t.Errorf("Summary = %q; want %q", got.Summary, want)
	}
}

// TestTemplateExecutorRespectsCancellation verifies that TemplateExecutor checks ctx before loading any file.
func TestTemplateExecutorRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exec := execution.TemplateExecutor{PromptsDir: t.TempDir()}
	_, err := exec.Analyze(ctx, execution.CodexRequest{Prompt: "findings"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Analyze() error = %v; want context.Canceled", err)
	}
}

