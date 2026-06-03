package chat_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
	internalchat "context-os/internal/chat"
)

// TestQueryRoutesSlackTodayToArtifacts verifies Slack today questions query local artifacts for the selected day.
func TestQueryRoutesSlackTodayToArtifacts(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-1",
		WorkspaceID: "ws1",
		Connector:   "slack",
		SourceURI:   "#team",
		Title:       "Daily release thread",
		Body:        "Release blocked by staging credentials.",
		IngestedAt:  time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}}}
	service := internalchat.NewService(fakeWorkspaces(), events, &fakeSyncRepository{})

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "give me today slack messages for #team",
		Timezone:    "UTC",
		LocalDate:   "2026-06-01",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Intent != "artifacts" {
		t.Fatalf("Intent = %q, want artifacts", result.Intent)
	}
	if result.Connector != "slack" {
		t.Fatalf("Connector = %q, want slack", result.Connector)
	}
	if result.SourceURI != "#team" {
		t.Fatalf("SourceURI = %q, want #team", result.SourceURI)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(result.Artifacts))
	}
	if events.lastQuery.Connector != "slack" {
		t.Fatalf("query connector = %q, want slack", events.lastQuery.Connector)
	}
	if events.lastQuery.Since == nil || events.lastQuery.Until == nil {
		t.Fatalf("date range was not set")
	}
	if got := events.lastQuery.Since.Format("2006-01-02"); got != "2026-06-01" {
		t.Fatalf("Since = %q, want 2026-06-01", got)
	}
}

// TestQuerySupportsNonSlackConnectors verifies source questions route to non-Slack connectors too.
func TestQuerySupportsNonSlackConnectors(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-2",
		WorkspaceID: "ws1",
		Connector:   "github",
		SourceURI:   "owner/repo",
		Title:       "Fix checkout flow",
		Body:        "Pull request merged after review.",
		IngestedAt:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}}}
	service := internalchat.NewService(fakeWorkspaces(), events, nil)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "show recent github issues",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Intent != "artifacts" {
		t.Fatalf("Intent = %q, want artifacts", result.Intent)
	}
	if result.Connector != "github" {
		t.Fatalf("Connector = %q, want github", result.Connector)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(result.Artifacts))
	}
}

// TestQueryInfersGitHubSourceFromConfiguredRepoName verifies repo slugs in messages constrain GitHub artifact queries to the matching synced source.
func TestQueryInfersGitHubSourceFromConfiguredRepoName(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{
		{
			ID:          "evt-mirofish",
			WorkspaceID: "ws1",
			Connector:   "github",
			SourceURI:   "sx-tane/MiroFish",
			Title:       "sx-tane/MiroFish",
			Body:        "Repository summary.",
			IngestedAt:  time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "evt-tourii",
			WorkspaceID: "ws1",
			Connector:   "github",
			SourceURI:   "sx-tane/tourii-backend",
			Title:       "sx-tane/tourii-backend",
			Body:        "Tourii backend repository summary.",
			IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
		},
	}}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/MiroFish", Status: "ready"},
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/tourii-backend", Status: "ready"},
	}}
	service := internalchat.NewService(fakeWorkspaces(), events, syncs)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "ok for tourii-backend can you give me more information",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.SourceURI != "sx-tane/tourii-backend" {
		t.Fatalf("SourceURI = %q, want sx-tane/tourii-backend", result.SourceURI)
	}
	if events.lastQuery.SourceURI != "sx-tane/tourii-backend" {
		t.Fatalf("query SourceURI = %q, want sx-tane/tourii-backend", events.lastQuery.SourceURI)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(result.Artifacts))
	}
	if strings.Contains(result.Answer, "MiroFish") {
		t.Fatalf("Answer = %q, should not include another repository", result.Answer)
	}
}

// TestQueryDoesNotPresentRepositoryArtifactAsLatestCommit verifies commit questions are honest when only repository-level GitHub data exists.
func TestQueryDoesNotPresentRepositoryArtifactAsLatestCommit(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-tourii",
		WorkspaceID: "ws1",
		Connector:   "github",
		SourceURI:   "sx-tane/tourii-backend",
		Title:       "sx-tane/tourii-backend",
		Body:        "Repository summary.",
		Metadata:    map[string]string{"object_type": "repository"},
		IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
	}}}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/tourii-backend", Status: "ready"},
	}}
	service := internalchat.NewService(fakeWorkspaces(), events, syncs)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "what is the latest commit for tourii-backend",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if !strings.Contains(result.Answer, "I do not have local commit artifacts") {
		t.Fatalf("Answer = %q, want commit artifact limitation", result.Answer)
	}
	if !strings.Contains(result.Answer, "sx-tane/tourii-backend") {
		t.Fatalf("Answer = %q, want requested repository scope", result.Answer)
	}
}

// TestQueryUsesLiveAnswererForGitHubCommitQuestions verifies commit questions can use a live Codex-backed source answer.
func TestQueryUsesLiveAnswererForGitHubCommitQuestions(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-tourii",
		WorkspaceID: "ws1",
		Connector:   "github",
		SourceURI:   "sx-tane/tourii-backend",
		Title:       "sx-tane/tourii-backend",
		Body:        "Repository summary.",
		Metadata:    map[string]string{"object_type": "repository"},
		IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
	}}}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/tourii-backend", Status: "ready"},
	}}
	live := &fakeLiveAnswerer{answer: "Latest commit is abc123 by sx-tane."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "what is the latest commit for tourii-backend",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Answer != "Latest commit is abc123 by sx-tane." {
		t.Fatalf("Answer = %q, want live answer", result.Answer)
	}
	if live.query.SourceURI != "sx-tane/tourii-backend" {
		t.Fatalf("live SourceURI = %q, want sx-tane/tourii-backend", live.query.SourceURI)
	}
}

// TestQueryUsesLiveAnswererForJiraLinksWithoutLocalArtifacts verifies pasted Jira links route to live Codex without requiring local artifacts.
func TestQueryUsesLiveAnswererForJiraLinksWithoutLocalArtifacts(t *testing.T) {
	events := &fakeEventRepository{}
	live := &fakeLiveAnswerer{answer: "BKGDEV-8457 is assigned to PMO review."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, &fakeSyncRepository{}, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "what is happening in https://acme.atlassian.net/browse/BKGDEV-8457",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Connector != "jira" {
		t.Fatalf("Connector = %q, want jira", result.Connector)
	}
	if live.query.SourceURI != "https://acme.atlassian.net/browse/BKGDEV-8457" {
		t.Fatalf("live SourceURI = %q, want Jira URL", live.query.SourceURI)
	}
	if events.lastQuery.SourceURI != "" {
		t.Fatalf("local query SourceURI = %q, want no local query before live success", events.lastQuery.SourceURI)
	}
}

// TestQueryUsesLiveAnswererForJiraIssueKeysWithoutLocalArtifacts verifies bare Jira issue keys route to live Codex without the word Jira.
func TestQueryUsesLiveAnswererForJiraIssueKeysWithoutLocalArtifacts(t *testing.T) {
	events := &fakeEventRepository{}
	live := &fakeLiveAnswerer{answer: "BKGDEV-8466 is in implementation review."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, &fakeSyncRepository{}, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "BKGDEV-8466 check this",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Connector != "jira" {
		t.Fatalf("Connector = %q, want jira", result.Connector)
	}
	if live.query.SourceURI != "BKGDEV-8466" {
		t.Fatalf("live SourceURI = %q, want BKGDEV-8466", live.query.SourceURI)
	}
}

// TestQueryKeepsExplicitRouteFieldsAheadOfMessageInference verifies frontend route fields win over ambiguous prompt text.
func TestQueryKeepsExplicitRouteFieldsAheadOfMessageInference(t *testing.T) {
	events := &fakeEventRepository{}
	live := &fakeLiveAnswerer{answer: "Explicit route answer."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, &fakeSyncRepository{}, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "BKGDEV-8466 check this github repo",
		Connector:   "jira",
		SourceURI:   "BKGDEV-8466",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if live.query.Connector != "jira" {
		t.Fatalf("live Connector = %q, want jira", live.query.Connector)
	}
	if live.query.SourceURI != "BKGDEV-8466" {
		t.Fatalf("live SourceURI = %q, want BKGDEV-8466", live.query.SourceURI)
	}
}

// TestQueryUsesLiveAnswererForSavedSourceName verifies saved connector syncs resolve named sources before live lookup.
func TestQueryUsesLiveAnswererForSavedSourceName(t *testing.T) {
	events := &fakeEventRepository{}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/tourii-backend", Status: "connected"},
	}}
	live := &fakeLiveAnswerer{answer: "tourii-backend has two open pull requests."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "for tourii-backend what is current",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Connector != "github" {
		t.Fatalf("Connector = %q, want github", result.Connector)
	}
	if live.query.SourceURI != "sx-tane/tourii-backend" {
		t.Fatalf("live SourceURI = %q, want sx-tane/tourii-backend", live.query.SourceURI)
	}
}

// TestQueryUsesConnectorLevelLiveAnswerer verifies an enabled plugin row can answer without a selected source URI.
func TestQueryUsesConnectorLevelLiveAnswerer(t *testing.T) {
	events := &fakeEventRepository{}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "github", Status: "connected"},
	}}
	live := &fakeLiveAnswerer{answer: "ContextOS repository is available through live GitHub."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "check context os repository",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Connector != "github" {
		t.Fatalf("Connector = %q, want github", result.Connector)
	}
	if live.query.SourceURI != "github" {
		t.Fatalf("live SourceURI = %q, want github", live.query.SourceURI)
	}
}

// TestQueryUsesLiveAnswererForOwnerRepoPrompt verifies owner/repo text routes to live GitHub through a saved connector row.
func TestQueryUsesLiveAnswererForOwnerRepoPrompt(t *testing.T) {
	events := &fakeEventRepository{}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "github", Status: "connected"},
	}}
	live := &fakeLiveAnswerer{answer: "sx-tane/context-os is available through live GitHub."}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "get sx-tane/context-os for me",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "codex" {
		t.Fatalf("Provider = %q, want codex", result.Provider)
	}
	if result.Connector != "github" {
		t.Fatalf("Connector = %q, want github", result.Connector)
	}
	if live.query.SourceURI != "sx-tane/context-os" {
		t.Fatalf("live SourceURI = %q, want sx-tane/context-os", live.query.SourceURI)
	}
}

// TestQueryKeepsFilesystemLocalFirst verifies filesystem questions do not route to live Codex.
func TestQueryKeepsFilesystemLocalFirst(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-file",
		WorkspaceID: "ws1",
		Connector:   "filesystem",
		SourceURI:   "docs/plan.md",
		Title:       "Local migration plan",
		Body:        "Use local artifacts as the source of truth for findings.",
		IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
	}}}
	live := &fakeLiveAnswerer{answer: "should not be used"}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, nil, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "summarize local file docs/plan.md",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "local" {
		t.Fatalf("Provider = %q, want local", result.Provider)
	}
	if live.calls != 0 {
		t.Fatalf("live calls = %d, want 0", live.calls)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(result.Artifacts))
	}
}

// TestQueryFallsBackToLocalArtifactsWhenLiveFails verifies live Codex failures are explicit before local artifact answers.
func TestQueryFallsBackToLocalArtifactsWhenLiveFails(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-github",
		WorkspaceID: "ws1",
		Connector:   "github",
		SourceURI:   "sx-tane/tourii-backend",
		Title:       "Repository fallback summary",
		Body:        "Local repository evidence is available.",
		IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
	}}}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "sx-tane/tourii-backend", Status: "connected"},
	}}
	live := &fakeLiveAnswerer{err: errors.New("plugin unavailable")}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "summarize tourii-backend",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Provider != "local" {
		t.Fatalf("Provider = %q, want local", result.Provider)
	}
	if !strings.Contains(result.Answer, "Live Codex lookup failed: plugin unavailable") {
		t.Fatalf("Answer = %q, want live failure prefix", result.Answer)
	}
	if !strings.Contains(result.Answer, "Repository fallback summary") {
		t.Fatalf("Answer = %q, want local fallback artifact", result.Answer)
	}
}

// TestQueryConnectorLevelLiveFallbackUsesConnectorArtifacts verifies connector-level live sources do not over-filter local fallback artifacts.
func TestQueryConnectorLevelLiveFallbackUsesConnectorArtifacts(t *testing.T) {
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-github",
		WorkspaceID: "ws1",
		Connector:   "github",
		SourceURI:   "sx-tane/tourii-backend",
		Title:       "Repository fallback summary",
		Body:        "Local repository evidence is available.",
		IngestedAt:  time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
	}}}
	syncs := &fakeSyncRepository{syncs: []repository.ConnectorSync{
		{WorkspaceID: "ws1", Connector: "github", SourceURI: "github", Status: "connected"},
	}}
	live := &fakeLiveAnswerer{err: errors.New("plugin unavailable")}
	service := internalchat.NewServiceWithLiveAnswerer(fakeWorkspaces(), events, syncs, live)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "summarize github",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if live.query.SourceURI != "github" {
		t.Fatalf("live SourceURI = %q, want github", live.query.SourceURI)
	}
	if events.lastQuery.SourceURI != "" {
		t.Fatalf("local query SourceURI = %q, want connector-wide fallback", events.lastQuery.SourceURI)
	}
	if !strings.Contains(result.Answer, "Repository fallback summary") {
		t.Fatalf("Answer = %q, want local fallback artifact", result.Answer)
	}
}

// TestQueryNoArtifactsDoesNotFallbackToFindings verifies source questions with no local data stay source-scoped.
func TestQueryNoArtifactsDoesNotFallbackToFindings(t *testing.T) {
	events := &fakeEventRepository{}
	service := internalchat.NewService(fakeWorkspaces(), events, nil)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "give me today jira tickets",
		Timezone:    "UTC",
		LocalDate:   "2026-06-01",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Intent != "artifacts" {
		t.Fatalf("Intent = %q, want artifacts", result.Intent)
	}
	if strings.Contains(strings.ToLower(result.Answer), "mismatch") {
		t.Fatalf("Answer = %q, should not mention mismatch fallback", result.Answer)
	}
	if result.Connector != "jira" {
		t.Fatalf("Connector = %q, want jira", result.Connector)
	}
}

// TestQueryCompactsVerboseSlackArtifacts verifies chat answers do not dump the
// entire ingested Slack artifact into the answer body.
func TestQueryCompactsVerboseSlackArtifacts(t *testing.T) {
	body := strings.Repeat("Channel metadata should not flood the chat answer. ", 20) + `
- JunQi Han: TODO includes BKGDEV-8457 and payment -l option support.
- YuXuan Yang asked whether to release PR 1551 together.
- JunQi Han said hotfix PR 1552 is under verification.
Message link: https://example.invalid/slack`
	events := &fakeEventRepository{events: []repository.IngestEvent{{
		ID:          "evt-verbose",
		WorkspaceID: "ws1",
		Connector:   "slack",
		SourceURI:   "#team",
		Title:       body,
		Body:        body,
		IngestedAt:  time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}}}
	service := internalchat.NewService(fakeWorkspaces(), events, nil)

	result, err := service.Query(context.Background(), internalchat.Query{
		WorkspaceID: "/workspace",
		Message:     "today slack message",
		Timezone:    "UTC",
		LocalDate:   "2026-06-01",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(result.Answer) > 700 {
		t.Fatalf("Answer length = %d, want compact answer: %s", len(result.Answer), result.Answer)
	}
	if strings.Contains(result.Answer, "Message link:") {
		t.Fatalf("Answer included raw evidence link line: %s", result.Answer)
	}
	if !strings.Contains(result.Answer, "Key points:") {
		t.Fatalf("Answer = %q, want key points", result.Answer)
	}
}

// fakeWorkspaces returns a workspace repository with one path-backed workspace.
func fakeWorkspaces() *fakeWorkspaceRepository {
	return &fakeWorkspaceRepository{workspace: repository.Workspace{
		ID:   "ws1",
		Name: "Workspace",
		Path: "/workspace",
	}}
}

type fakeWorkspaceRepository struct {
	workspace repository.Workspace
}

func (f *fakeWorkspaceRepository) Upsert(ctx context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	return workspace, nil
}

func (f *fakeWorkspaceRepository) GetByPath(ctx context.Context, path string) (*repository.Workspace, error) {
	if path != f.workspace.Path {
		return nil, nil
	}
	workspace := f.workspace
	return &workspace, nil
}

func (f *fakeWorkspaceRepository) List(ctx context.Context) ([]repository.Workspace, error) {
	return []repository.Workspace{f.workspace}, nil
}

type fakeEventRepository struct {
	events    []repository.IngestEvent
	lastQuery repository.EventQuery
}

func (f *fakeEventRepository) UpsertBatch(ctx context.Context, workspaceID string, events []repository.IngestEvent) (int, error) {
	return len(events), nil
}

func (f *fakeEventRepository) ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]repository.IngestEvent, error) {
	return f.events, nil
}

func (f *fakeEventRepository) Query(ctx context.Context, workspaceID string, query repository.EventQuery) ([]repository.IngestEvent, error) {
	f.lastQuery = query
	if len(f.events) == 0 {
		return nil, nil
	}
	out := make([]repository.IngestEvent, 0, len(f.events))
	for _, event := range f.events {
		if query.Connector != "" && event.Connector != query.Connector {
			continue
		}
		if query.SourceURI != "" && event.SourceURI != query.SourceURI {
			continue
		}
		out = append(out, event)
	}
	return out, nil
}

func (f *fakeEventRepository) Count(ctx context.Context, workspaceID, connector string) (int, error) {
	return len(f.events), nil
}

type fakeSyncRepository struct {
	syncs []repository.ConnectorSync
}

func (f *fakeSyncRepository) Upsert(ctx context.Context, sync repository.ConnectorSync) error {
	return nil
}

func (f *fakeSyncRepository) Get(ctx context.Context, workspaceID, connector, sourceURI string) (*repository.ConnectorSync, error) {
	return nil, nil
}

func (f *fakeSyncRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	if len(f.syncs) > 0 {
		return f.syncs, nil
	}
	return []repository.ConnectorSync{{WorkspaceID: workspaceID, Connector: "slack", SourceURI: "#team", Status: "idle"}}, nil
}

var _ repository.WorkspaceRepository = (*fakeWorkspaceRepository)(nil)
var _ repository.EventRepository = (*fakeEventRepository)(nil)
var _ repository.SyncRepository = (*fakeSyncRepository)(nil)
var _ internalchat.LiveAnswerer = (*fakeLiveAnswerer)(nil)
var _ repository.EntityRepository = (*unusedEntityRepository)(nil)

type fakeLiveAnswerer struct {
	answer string
	err    error
	calls  int
	query  internalchat.LiveQuery
}

func (f *fakeLiveAnswerer) Answer(ctx context.Context, query internalchat.LiveQuery) (string, error) {
	f.calls += 1
	f.query = query
	if f.err != nil {
		return "", f.err
	}
	return f.answer, nil
}

type unusedEntityRepository struct{}

func (u *unusedEntityRepository) UpsertEntities(ctx context.Context, workspaceID string, canonical []entities.CanonicalEntity) error {
	return nil
}

func (u *unusedEntityRepository) UpsertRelationships(ctx context.Context, workspaceID string, rels []types.Relationship) error {
	return nil
}

func (u *unusedEntityRepository) ListEntities(ctx context.Context, workspaceID, entityType string) ([]entities.CanonicalEntity, error) {
	return nil, nil
}

func (u *unusedEntityRepository) ListRelationships(ctx context.Context, workspaceID string, entityIDs []string) ([]types.Relationship, error) {
	return nil, nil
}
