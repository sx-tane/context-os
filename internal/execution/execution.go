package execution

import "context" // provides cancellation and deadline support for analysis calls

// CodexRequest carries the prompt and supporting context for an AI analysis task.
type CodexRequest struct {
	Prompt  string            `json:"prompt"`  // the question or instruction sent to the AI executor
	Context map[string]string `json:"context"` // additional key-value data the executor may use when building its response
}

// CodexResult holds the output produced by an AI analysis task.
type CodexResult struct {
	Summary  string            `json:"summary"`  // human-readable conclusion from the analysis
	Metadata map[string]string `json:"metadata"` // structured details about how the result was produced
}

// CodexExecutor is the interface any AI execution backend must satisfy.
type CodexExecutor interface {
	Analyze(context.Context, CodexRequest) (CodexResult, error) // runs an analysis task and returns a result
}

// LocalStubExecutor is a no-op implementation used until a real executor is wired in.
type LocalStubExecutor struct{}

// Analyze returns a placeholder result without calling any external AI service.
func (LocalStubExecutor) Analyze(ctx context.Context, req CodexRequest) (CodexResult, error) {
	if err := ctx.Err(); err != nil {
		return CodexResult{}, err // respect cancellation before doing any work
	}
	return CodexResult{
		Summary:  "Codex execution is prepared for local hidden orchestration.",          // placeholder message indicating the stub ran
		Metadata: map[string]string{"mode": "local-stub", "prompt": req.Prompt},          // record how this result was produced and what prompt was used
	}, nil
}
