package presentation

// White-box tests inspect handler internals to verify option wiring after the file split.

import (
	"context"
	"testing"

	"context-os/internal/stages/execution"
)

type optionExecutor struct{}

func (optionExecutor) Analyze(context.Context, execution.CodexRequest) (execution.CodexResult, error) {
	return execution.CodexResult{Summary: "configured"}, nil
}

// TestNewHandlerOptionWiring verifies handler options retain configured optional dependencies.
func TestNewHandlerOptionWiring(t *testing.T) {
	t.Parallel()

	executor := optionExecutor{}
	handler := NewHandler(nil, nil, nil, nil, nil, WithExecutor(executor))

	if handler.executor == nil {
		t.Fatal("executor = nil, want configured executor")
	}
	if _, ok := handler.executor.(optionExecutor); !ok {
		t.Fatalf("executor type = %T, want optionExecutor", handler.executor)
	}
}
