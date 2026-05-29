package execution_test

import (
	"context"
	"errors"
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
