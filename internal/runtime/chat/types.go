package chat

import (
	"context"
	"errors"
	"regexp"
	"time"

	"context-os/domain/repository"
)

const (
	intentArtifacts   = "artifacts"
	intentFindings    = "findings"
	intentStatus      = "status"
	intentUnsupported = "unsupported"
	modeAuto          = "auto"
	modeCodex         = "codex"
	modeLocal         = "local"
	defaultLimit      = 20
	maxLimit          = 100
)

var (
	// ErrWorkspaceRequired is returned when a chat query omits workspace scope.
	ErrWorkspaceRequired = errors.New("workspace is required")
	// ErrWorkspaceNotFound is returned when a chat query references an unknown workspace.
	ErrWorkspaceNotFound = errors.New("workspace not found")
	// ErrMessageRequired is returned when a chat query omits the user message.
	ErrMessageRequired = errors.New("message is required")

	sourcePattern      = regexp.MustCompile(`(?i)(#[A-Za-z0-9_.-]+|[a-z]+://[^\s,]+|https?://[^\s,]+|[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`)
	sourceTokenPattern = regexp.MustCompile(`[^a-z0-9_.-]+`)
	sourceSubTokenPat  = regexp.MustCompile(`[-_.]+`)
	githubRepoPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	jiraKeyPattern     = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
)

// Query is one local chat request.
type Query struct {
	WorkspaceID      string
	WorkspacePath    string
	Message          string
	Connector        string
	Connectors       []string
	SourceURI        string
	Mode             string
	Timezone         string
	LocalDate        string
	ResponseLanguage string
	Limit            int
	Progress         func(string)
}

// Result is one deterministic local chat answer.
type Result struct {
	Intent         string
	WorkspaceID    string
	WorkspacePath  string
	Connector      string
	SourceURI      string
	Provider       string
	Answer         string
	Summary        string
	AnswerSections []AnswerSection
	Since          *time.Time
	Until          *time.Time
	Artifacts      []repository.IngestEvent
	Syncs          []repository.ConnectorSync
}

// AnswerSection is one structured source-backed section in a chat answer.
type AnswerSection struct {
	SourceLabel string
	Connector   string
	SourceURI   string
	Summary     string
	Facts       []string
	OpenItems   []string
	CodingNotes []string
	Links       []string
	Timestamps  []string
	Confidence  float64
	Status      string
}

// LiveQuery is one optional live source question routed through Codex-backed connectors.
type LiveQuery struct {
	WorkspaceID      string
	WorkspacePath    string
	Connector        string
	SourceURI        string
	Message          string
	ResponseLanguage string
	Progress         func(string)
}

// LiveAnswerer answers a source question from a live connector account.
type LiveAnswerer interface {
	Answer(ctx context.Context, query LiveQuery) (string, error)
}

// LiveSessionResetter clears workspace-scoped live chat conversation metadata.
type LiveSessionResetter interface {
	ResetSession(ctx context.Context, workspaceID string) error
}

// liveFanoutResult carries one live lookup result back from a fanout worker.
type liveFanoutResult struct {
	index     int
	connector string
	sourceURI string
	sections  []AnswerSection
	failure   string
}
