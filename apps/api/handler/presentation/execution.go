package presentation

import (
	"context"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/types"
	"context-os/internal/stages/execution"
	stagepresentation "context-os/internal/stages/presentation"
	"strconv"
	"strings"
	"time"
)

// runAssistiveExecution invokes the configured executor and wraps the result as assistive evidence.
func (h *Handler) runAssistiveExecution(ctx context.Context, traceID string, req request.PresentationFindings, role stagepresentation.Role, mismatchIDs []string, mismatches []types.Mismatch) response.ExecutionEvidence {
	execCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	exec := h.executor
	if exec == nil {
		exec = execution.LocalStubExecutor{}
	}

	prompt := "findings"
	result, err := exec.Analyze(execCtx, execution.CodexRequest{
		Prompt: prompt,
		Context: map[string]string{
			"trace_id":       traceID,
			"connector":      strings.ToLower(strings.TrimSpace(req.Connector)),
			"uri":            strings.TrimSpace(req.URI),
			"role":           string(role),
			"mismatch_count": strconv.Itoa(len(mismatches)),
			"mismatch_ids":   strings.Join(mismatchIDs, ","),
		},
	})

	metadata := map[string]string{"trace_id": traceID, "mode": "local-stub"}
	if err != nil {
		metadata["status"] = "error"
		return response.ExecutionEvidence{
			Enabled:   true,
			Assistive: true,
			Summary:   "assistive execution failed",
			Metadata:  metadata,
			Error:     err.Error(),
		}
	}
	for key, value := range result.Metadata {
		metadata[key] = value
	}
	metadata["status"] = "ok"
	return response.ExecutionEvidence{
		Enabled:   true,
		Assistive: true,
		Summary:   result.Summary,
		Metadata:  metadata,
	}
}
