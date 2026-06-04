package execution

import (
	"context" // provides cancellation and deadline support for analysis calls
	"fmt"     // formats substituted prompt output
	"os"      // reads prompt template files from disk
	"strings" // variable substitution inside templates
)

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
		Summary:  "Codex execution is prepared for local hidden orchestration.",
		Metadata: map[string]string{"mode": "local-stub", "prompt": req.Prompt},
	}, nil
}

// TemplateExecutor loads a Markdown prompt template from the prompts/ directory,
// substitutes {{key}} placeholders with values from the request context, and
// returns the rendered template as the summary. This makes the prompt content
// visible during development and tests while a real AI backend is not yet wired.
//
// Template file resolution: prompts/<req.Prompt>.md relative to promptsDir.
// Unknown variables are left as-is so callers can inspect them.
type TemplateExecutor struct {
	// PromptsDir is the directory that contains *.md prompt template files.
	// If empty, it defaults to "prompts".
	PromptsDir string
}

// Analyze loads the template for req.Prompt, substitutes req.Context variables,
// and returns the rendered content as CodexResult.Summary.
func (e TemplateExecutor) Analyze(ctx context.Context, req CodexRequest) (CodexResult, error) {
	if err := ctx.Err(); err != nil {
		return CodexResult{}, err
	}
	dir := e.PromptsDir
	if dir == "" {
		dir = "prompts"
	}
	path := fmt.Sprintf("%s/%s.md", dir, req.Prompt)
	raw, err := os.ReadFile(path) // #nosec G304 — path is built from a safe constant prefix
	if err != nil {
		// Fall back to the stub when the template file is missing so callers are
		// never broken by a missing file during development or tests.
		return CodexResult{
			Summary:  fmt.Sprintf("template not found: %s (stub mode)", path),
			Metadata: map[string]string{"mode": "template-stub", "prompt": req.Prompt, "path": path},
		}, nil
	}
	rendered := applyContext(string(raw), req.Context)
	return CodexResult{
		Summary: rendered,
		Metadata: map[string]string{
			"mode":   "template",
			"prompt": req.Prompt,
			"path":   path,
		},
	}, nil
}

// applyContext replaces every {{key}} occurrence in tmpl with the matching value
// from vars. Keys not present in vars are left unreplaced.
func applyContext(tmpl string, vars map[string]string) string {
	for key, value := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{{"+key+"}}", value)
	}
	return tmpl
}
