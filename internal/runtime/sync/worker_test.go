package sync

import (
	"context"
	"reflect"
	"testing"
	"time"

	"context-os/domain/repository"
)

type fakeWorkspaceRepo struct {
	workspaces []repository.Workspace
	err        error
}

func (f fakeWorkspaceRepo) Upsert(context.Context, repository.Workspace) (repository.Workspace, error) {
	panic("unexpected call")
}

func (f fakeWorkspaceRepo) GetByPath(context.Context, string) (*repository.Workspace, error) {
	panic("unexpected call")
}

func (f fakeWorkspaceRepo) List(context.Context) ([]repository.Workspace, error) {
	return append([]repository.Workspace(nil), f.workspaces...), f.err
}

type fakeSyncRepo struct {
	syncs     map[string][]repository.ConnectorSync
	upserted  []repository.ConnectorSync
	listErr   error
	upsertErr error
}

func (f *fakeSyncRepo) Upsert(_ context.Context, s repository.ConnectorSync) error {
	f.upserted = append(f.upserted, s)
	if f.upsertErr != nil {
		return f.upsertErr
	}
	key := s.WorkspaceID
	f.syncs[key] = append(f.syncs[key], s)
	return nil
}

func (f *fakeSyncRepo) Get(context.Context, string, string, string) (*repository.ConnectorSync, error) {
	panic("unexpected call")
}

func (f *fakeSyncRepo) ListByWorkspace(_ context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]repository.ConnectorSync(nil), f.syncs[workspaceID]...), nil
}

type fakeEventRepo struct{}

func (fakeEventRepo) UpsertBatch(context.Context, string, []repository.IngestEvent) (int, error) {
	panic("unexpected call")
}

func (fakeEventRepo) ListByWorkspace(context.Context, string, string, int) ([]repository.IngestEvent, error) {
	panic("unexpected call")
}

func (fakeEventRepo) Query(context.Context, string, repository.EventQuery) ([]repository.IngestEvent, error) {
	panic("unexpected call")
}

func (fakeEventRepo) Count(context.Context, string, string) (int, error) {
	panic("unexpected call")
}

// TestRunPassMarksStaleAndErroredSyncsPending verifies stale or errored syncs are reset to pending.
func TestRunPassMarksStaleAndErroredSyncsPending(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	stale := now.Add(-30 * time.Minute)
	fresh := now.Add(-5 * time.Minute)

	workspaces := fakeWorkspaceRepo{
		workspaces: []repository.Workspace{{ID: "ws-1"}},
	}
	syncs := &fakeSyncRepo{
		syncs: map[string][]repository.ConnectorSync{
			"ws-1": {
				{
					WorkspaceID:  "ws-1",
					Connector:    "github",
					SourceURI:    "repo",
					Status:       "connected",
					LastSyncedAt: &stale,
				},
				{
					WorkspaceID:  "ws-1",
					Connector:    "slack",
					SourceURI:    "channel",
					Status:       "error",
					LastSyncedAt: &fresh,
				},
				{
					WorkspaceID:  "ws-1",
					Connector:    "notion",
					SourceURI:    "page",
					Status:       "connected",
					LastSyncedAt: &fresh,
				},
			},
		},
	}

	worker := NewWorker(workspaces, syncs, fakeEventRepo{})
	worker.runPass(context.Background())

	if len(syncs.upserted) != 2 {
		t.Fatalf("len(upserted) = %d, want %d", len(syncs.upserted), 2)
	}
	got := []string{syncs.upserted[0].Connector, syncs.upserted[1].Connector}
	want := []string{"github", "slack"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("upserted connectors = %#v, want %#v", got, want)
	}
	for _, s := range syncs.upserted {
		if s.Status != "pending" {
			t.Errorf("status for %s = %q, want %q", s.Connector, s.Status, "pending")
		}
	}
}

// TestRunPassKeepsSavedConnectedSourcesReady verifies saved live source references are not marked pending before first ingest.
func TestRunPassKeepsSavedConnectedSourcesReady(t *testing.T) {
	t.Parallel()

	workspaces := fakeWorkspaceRepo{
		workspaces: []repository.Workspace{{ID: "ws-1"}},
	}
	syncs := &fakeSyncRepo{
		syncs: map[string][]repository.ConnectorSync{
			"ws-1": {
				{
					WorkspaceID: "ws-1",
					Connector:   "github",
					SourceURI:   "repo",
					Status:      "connected",
				},
			},
		},
	}

	worker := NewWorker(workspaces, syncs, fakeEventRepo{})
	worker.runPass(context.Background())

	if len(syncs.upserted) != 0 {
		t.Fatalf("len(upserted) = %d, want %d", len(syncs.upserted), 0)
	}
}

// TestRunPassMarksSyncRowsWithLocalStatePending verifies never-finished local sync rows still require attention.
func TestRunPassMarksSyncRowsWithLocalStatePending(t *testing.T) {
	t.Parallel()

	workspaces := fakeWorkspaceRepo{
		workspaces: []repository.Workspace{{ID: "ws-1"}},
	}
	syncs := &fakeSyncRepo{
		syncs: map[string][]repository.ConnectorSync{
			"ws-1": {
				{
					WorkspaceID: "ws-1",
					Connector:   "slack",
					SourceURI:   "#team",
					Status:      "connected",
					Cursor:      "cursor-1",
				},
			},
		},
	}

	worker := NewWorker(workspaces, syncs, fakeEventRepo{})
	worker.runPass(context.Background())

	if len(syncs.upserted) != 1 {
		t.Fatalf("len(upserted) = %d, want %d", len(syncs.upserted), 1)
	}
	if syncs.upserted[0].Status != "pending" {
		t.Fatalf("Status = %q, want pending", syncs.upserted[0].Status)
	}
}

// TestRunPassSkipsEmptyWorkspaceLists verifies the worker exits without touching syncs when no workspaces exist.
func TestRunPassSkipsEmptyWorkspaceLists(t *testing.T) {
	t.Parallel()

	workspaces := fakeWorkspaceRepo{}
	syncs := &fakeSyncRepo{syncs: map[string][]repository.ConnectorSync{}}

	worker := NewWorker(workspaces, syncs, fakeEventRepo{})
	worker.runPass(context.Background())

	if len(syncs.upserted) != 0 {
		t.Fatalf("len(upserted) = %d, want %d", len(syncs.upserted), 0)
	}
}
