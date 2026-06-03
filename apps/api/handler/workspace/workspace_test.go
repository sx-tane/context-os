package workspace_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	workspacehandler "context-os/apps/api/handler/workspace"
	"context-os/domain/repository"
)

// TestListReturnsSnakeCaseWorkspaceFields verifies workspace lists use stable JSON response field names.
func TestListReturnsSnakeCaseWorkspaceFields(t *testing.T) {
	timestamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	handler := workspacehandler.NewHandler(workspaceRepo{workspaces: []repository.Workspace{
		{ID: "workspace", Name: "workspace", Path: "/workspace", CreatedAt: timestamp, UpdatedAt: timestamp},
	}}, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/workspace", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeObject(t, rec.Body.Bytes())
	workspaces := objectSlice(t, body, "workspaces")
	if len(workspaces) != 1 {
		t.Fatalf("len(workspaces) = %d, want 1", len(workspaces))
	}
	if _, ok := workspaces[0]["Path"]; ok {
		t.Fatalf("workspaces[0][Path] present, want snake_case path only")
	}
	if workspaces[0]["path"] != "/workspace" {
		t.Fatalf("workspaces[0][path] = %v, want /workspace", workspaces[0]["path"])
	}
}

// TestStatusReturnsSnakeCaseWorkspaceFields verifies workspace status uses stable JSON field names for workspace and sync state.
func TestStatusReturnsSnakeCaseWorkspaceFields(t *testing.T) {
	timestamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	workspace := repository.Workspace{ID: "workspace", Name: "workspace", Path: "/workspace", CreatedAt: timestamp, UpdatedAt: timestamp}
	handler := workspacehandler.NewHandler(
		workspaceRepo{workspaces: []repository.Workspace{workspace}},
		eventRepo{count: 7},
		nil,
		nil,
		syncRepo{syncs: []repository.ConnectorSync{
			{WorkspaceID: "workspace", Connector: "github", SourceURI: "github://repo", Cursor: "cursor-1", LastSyncedAt: &timestamp, EventCount: 3, Status: "ready"},
		}},
	)

	req := httptest.NewRequest(http.MethodGet, "/workspace/status?path=/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeObject(t, rec.Body.Bytes())
	workspaceBody := objectField(t, body, "workspace")
	if _, ok := workspaceBody["Path"]; ok {
		t.Fatalf("workspace[Path] present, want snake_case path only")
	}
	if workspaceBody["path"] != "/workspace" {
		t.Fatalf("workspace[path] = %v, want /workspace", workspaceBody["path"])
	}
	if body["event_count"] != float64(7) {
		t.Fatalf("event_count = %v, want 7", body["event_count"])
	}
	syncs := objectSlice(t, body, "syncs")
	if len(syncs) != 1 {
		t.Fatalf("len(syncs) = %d, want 1", len(syncs))
	}
	if _, ok := syncs[0]["SourceURI"]; ok {
		t.Fatalf("syncs[0][SourceURI] present, want snake_case source_uri only")
	}
	if syncs[0]["source_uri"] != "github://repo" {
		t.Fatalf("syncs[0][source_uri] = %v, want github://repo", syncs[0]["source_uri"])
	}
}

// TestDeleteRejectsMissingPath verifies workspace delete requires a path query parameter.
func TestDeleteRejectsMissingPath(t *testing.T) {
	handler := workspacehandler.NewHandler(&resettableWorkspaceRepo{}, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := decodeObject(t, rec.Body.Bytes())
	if body["error"] != "invalid_request" {
		t.Fatalf("error = %v, want invalid_request", body["error"])
	}
}

// TestDeleteRemovesWorkspaceWithoutRecreatingRow verifies workspace delete calls DeleteByPath and does not upsert a replacement row.
func TestDeleteRemovesWorkspaceWithoutRecreatingRow(t *testing.T) {
	repo := &resettableWorkspaceRepo{}
	handler := workspacehandler.NewHandler(repo, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/workspace?path=/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(repo.deletedPaths) != 1 || repo.deletedPaths[0] != "/workspace" {
		t.Fatalf("deletedPaths = %v, want [/workspace]", repo.deletedPaths)
	}
	if repo.upsertCount != 0 {
		t.Fatalf("upsertCount = %d, want 0", repo.upsertCount)
	}
	body := decodeObject(t, rec.Body.Bytes())
	if body["deleted"] != true {
		t.Fatalf("deleted = %v, want true", body["deleted"])
	}
}

// TestDeleteMissingWorkspaceSucceeds verifies deleting an unknown workspace path is a successful no-op.
func TestDeleteMissingWorkspaceSucceeds(t *testing.T) {
	repo := &resettableWorkspaceRepo{}
	handler := workspacehandler.NewHandler(repo, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/workspace?path=/missing", nil)
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(repo.deletedPaths) != 1 || repo.deletedPaths[0] != "/missing" {
		t.Fatalf("deletedPaths = %v, want [/missing]", repo.deletedPaths)
	}
}

// TestSourceRegistersConnectedSource verifies workspace source registration saves a connected sync row without ingest state.
func TestSourceRegistersConnectedSource(t *testing.T) {
	workspace := repository.Workspace{ID: "workspace", Name: "workspace", Path: "/workspace"}
	syncs := &recordingSyncRepo{}
	handler := workspacehandler.NewHandler(
		workspaceRepo{workspaces: []repository.Workspace{workspace}},
		nil,
		nil,
		nil,
		syncs,
	)
	body := []byte(`{"workspace_id":"/workspace","connector":"github","source_uri":"context-os/context-os"}`)

	req := httptest.NewRequest(http.MethodPost, "/workspace/source", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Source(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(syncs.upserts) != 1 {
		t.Fatalf("upsert count = %d, want 1", len(syncs.upserts))
	}
	sync := syncs.upserts[0]
	if sync.WorkspaceID != "workspace" {
		t.Fatalf("WorkspaceID = %q, want workspace", sync.WorkspaceID)
	}
	if sync.Connector != "github" {
		t.Fatalf("Connector = %q, want github", sync.Connector)
	}
	if sync.SourceURI != "context-os/context-os" {
		t.Fatalf("SourceURI = %q, want context-os/context-os", sync.SourceURI)
	}
	if sync.Status != "connected" {
		t.Fatalf("Status = %q, want connected", sync.Status)
	}
	if sync.EventCount != 0 {
		t.Fatalf("EventCount = %d, want 0", sync.EventCount)
	}
	if sync.LastSyncedAt != nil {
		t.Fatalf("LastSyncedAt = %v, want nil", sync.LastSyncedAt)
	}
	responseBody := decodeObject(t, rec.Body.Bytes())
	if responseBody["status"] != "connected" {
		t.Fatalf("response status = %v, want connected", responseBody["status"])
	}
	if responseBody["event_count"] != float64(0) {
		t.Fatalf("response event_count = %v, want 0", responseBody["event_count"])
	}
}

// TestSourceCreatesWorkspaceWhenNeeded verifies source registration upserts a workspace before saving sync state.
func TestSourceCreatesWorkspaceWhenNeeded(t *testing.T) {
	workspaces := &recordingWorkspaceRepo{}
	syncs := &recordingSyncRepo{}
	handler := workspacehandler.NewHandler(workspaces, nil, nil, nil, syncs)
	body := []byte(`{"workspace_id":"/new/project","connector":"jira","source_uri":"https://acme.atlassian.net/browse/ABC-1"}`)

	req := httptest.NewRequest(http.MethodPost, "/workspace/source", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Source(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(workspaces.upserts) != 1 {
		t.Fatalf("workspace upserts = %d, want 1", len(workspaces.upserts))
	}
	if workspaces.upserts[0].Path != "/new/project" {
		t.Fatalf("workspace Path = %q, want /new/project", workspaces.upserts[0].Path)
	}
	if workspaces.upserts[0].Name != "project" {
		t.Fatalf("workspace Name = %q, want project", workspaces.upserts[0].Name)
	}
	if len(syncs.upserts) != 1 {
		t.Fatalf("sync upserts = %d, want 1", len(syncs.upserts))
	}
	if syncs.upserts[0].WorkspaceID != "created-workspace" {
		t.Fatalf("sync WorkspaceID = %q, want created-workspace", syncs.upserts[0].WorkspaceID)
	}
}

// TestSourceRejectsMissingRequiredFields verifies source registration rejects incomplete connected-source requests.
func TestSourceRejectsMissingRequiredFields(t *testing.T) {
	handler := workspacehandler.NewHandler(workspaceRepo{}, nil, nil, nil, &recordingSyncRepo{})
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "workspace", body: `{"connector":"github","source_uri":"owner/repo"}`, want: "workspace_id is required"},
		{name: "connector", body: `{"workspace_id":"/workspace","source_uri":"owner/repo"}`, want: "connector is required"},
		{name: "source", body: `{"workspace_id":"/workspace","connector":"github"}`, want: "source_uri is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/workspace/source", bytes.NewReader([]byte(tc.body)))
			rec := httptest.NewRecorder()
			handler.Source(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			body := decodeObject(t, rec.Body.Bytes())
			if body["message"] != tc.want {
				t.Errorf("message = %v, want %q", body["message"], tc.want)
			}
		})
	}
}

type workspaceRepo struct {
	workspaces []repository.Workspace
}

func (r workspaceRepo) Upsert(_ context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	return workspace, nil
}

func (r workspaceRepo) GetByPath(_ context.Context, path string) (*repository.Workspace, error) {
	for _, workspace := range r.workspaces {
		if workspace.Path == path {
			found := workspace
			return &found, nil
		}
	}
	return nil, nil
}

func (r workspaceRepo) List(_ context.Context) ([]repository.Workspace, error) {
	return r.workspaces, nil
}

type eventRepo struct {
	count int
}

func (r eventRepo) UpsertBatch(_ context.Context, _ string, _ []repository.IngestEvent) (int, error) {
	return 0, nil
}

func (r eventRepo) ListByWorkspace(_ context.Context, _, _ string, _ int) ([]repository.IngestEvent, error) {
	return nil, nil
}

func (r eventRepo) Query(_ context.Context, _ string, _ repository.EventQuery) ([]repository.IngestEvent, error) {
	return nil, nil
}

func (r eventRepo) Count(_ context.Context, _, _ string) (int, error) {
	return r.count, nil
}

type syncRepo struct {
	syncs []repository.ConnectorSync
}

func (r syncRepo) Upsert(_ context.Context, _ repository.ConnectorSync) error {
	return nil
}

func (r syncRepo) Get(_ context.Context, _, _, _ string) (*repository.ConnectorSync, error) {
	return nil, nil
}

func (r syncRepo) ListByWorkspace(_ context.Context, _ string) ([]repository.ConnectorSync, error) {
	return r.syncs, nil
}

type recordingSyncRepo struct {
	syncs   []repository.ConnectorSync
	upserts []repository.ConnectorSync
}

func (r *recordingSyncRepo) Upsert(_ context.Context, sync repository.ConnectorSync) error {
	r.upserts = append(r.upserts, sync)
	return nil
}

func (r *recordingSyncRepo) Get(_ context.Context, _, _, _ string) (*repository.ConnectorSync, error) {
	return nil, nil
}

func (r *recordingSyncRepo) ListByWorkspace(_ context.Context, _ string) ([]repository.ConnectorSync, error) {
	return r.syncs, nil
}

type resettableWorkspaceRepo struct {
	workspaceRepo
	deletedPaths []string
	upsertCount  int
}

func (r *resettableWorkspaceRepo) Upsert(_ context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	r.upsertCount += 1
	return workspace, nil
}

func (r *resettableWorkspaceRepo) DeleteByPath(_ context.Context, path string) error {
	r.deletedPaths = append(r.deletedPaths, path)
	r.workspaces = r.workspaces[:0]
	return nil
}

type recordingWorkspaceRepo struct {
	workspaceRepo
	upserts []repository.Workspace
}

func (r *recordingWorkspaceRepo) Upsert(_ context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	workspace.ID = "created-workspace"
	r.upserts = append(r.upserts, workspace)
	r.workspaces = append(r.workspaces, workspace)
	return workspace, nil
}

// decodeObject decodes a JSON object from body.
func decodeObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return decoded
}

// objectField returns a named JSON object field.
func objectField(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T, want object", key, parent[key])
	}
	return value
}

// objectSlice returns a named JSON object slice field.
func objectSlice(t *testing.T, parent map[string]any, key string) []map[string]any {
	t.Helper()
	items, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s = %T, want array", key, parent[key])
	}
	objects := make([]map[string]any, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s item = %T, want object", key, item)
		}
		objects = append(objects, object)
	}
	return objects
}
