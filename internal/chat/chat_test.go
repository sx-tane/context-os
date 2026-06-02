package chat_test

import (
	"context"
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

type fakeSyncRepository struct{}

func (f *fakeSyncRepository) Upsert(ctx context.Context, sync repository.ConnectorSync) error {
	return nil
}

func (f *fakeSyncRepository) Get(ctx context.Context, workspaceID, connector, sourceURI string) (*repository.ConnectorSync, error) {
	return nil, nil
}

func (f *fakeSyncRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	return []repository.ConnectorSync{{WorkspaceID: workspaceID, Connector: "slack", SourceURI: "#team", Status: "idle"}}, nil
}

var _ repository.WorkspaceRepository = (*fakeWorkspaceRepository)(nil)
var _ repository.EventRepository = (*fakeEventRepository)(nil)
var _ repository.SyncRepository = (*fakeSyncRepository)(nil)
var _ repository.EntityRepository = (*unusedEntityRepository)(nil)

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
