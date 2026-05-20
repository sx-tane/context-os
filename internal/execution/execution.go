package execution

import "context"

type CodexRequest struct {
	Prompt  string            `json:"prompt"`
	Context map[string]string `json:"context"`
}

type CodexResult struct {
	Summary  string            `json:"summary"`
	Metadata map[string]string `json:"metadata"`
}

type CodexExecutor interface {
	Analyze(context.Context, CodexRequest) (CodexResult, error)
}

type LocalStubExecutor struct{}

func (LocalStubExecutor) Analyze(ctx context.Context, req CodexRequest) (CodexResult, error) {
	if err := ctx.Err(); err != nil {
		return CodexResult{}, err
	}
	return CodexResult{
		Summary:  "Codex execution is prepared for local hidden orchestration.",
		Metadata: map[string]string{"mode": "local-stub", "prompt": req.Prompt},
	}, nil
}
