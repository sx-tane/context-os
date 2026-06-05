package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"context-os/domain/repository"
)

type fakeWorkspaceRepo struct {
	workspaces []repository.Workspace
	getByPath  map[string]*repository.Workspace
	listErr    error
	getErr     error
}

func (f fakeWorkspaceRepo) Upsert(context.Context, repository.Workspace) (repository.Workspace, error) {
	panic("unexpected call")
}

func (f fakeWorkspaceRepo) GetByPath(_ context.Context, path string) (*repository.Workspace, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if workspace, ok := f.getByPath[path]; ok {
		return workspace, nil
	}
	return nil, nil
}

func (f fakeWorkspaceRepo) List(context.Context) ([]repository.Workspace, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]repository.Workspace(nil), f.workspaces...), nil
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

type cleanupEventRepo struct {
	events  []repository.IngestEvent
	deleted []string
}

func (c *cleanupEventRepo) UpsertBatch(context.Context, string, []repository.IngestEvent) (int, error) {
	panic("unexpected call")
}

func (c *cleanupEventRepo) ListByWorkspace(context.Context, string, string, int) ([]repository.IngestEvent, error) {
	panic("unexpected call")
}

func (c *cleanupEventRepo) Query(context.Context, string, repository.EventQuery) ([]repository.IngestEvent, error) {
	return append([]repository.IngestEvent(nil), c.events...), nil
}

func (c *cleanupEventRepo) Count(context.Context, string, string) (int, error) {
	panic("unexpected call")
}

func (c *cleanupEventRepo) DeleteByIDs(_ context.Context, _ string, ids []string) (int, error) {
	c.deleted = append([]string(nil), ids...)
	return len(ids), nil
}

type fakeGraphEvidenceDeleter struct {
	deleted []string
	result  repository.GraphCleanupResult
	err     error
}

func (f *fakeGraphEvidenceDeleter) DeleteGraphEvidenceByEventIDs(_ context.Context, _ string, ids []string) (repository.GraphCleanupResult, error) {
	f.deleted = append([]string(nil), ids...)
	if f.err != nil {
		return repository.GraphCleanupResult{}, f.err
	}
	return f.result, nil
}

// TestParseLimit verifies query limits default, clamp, and validate as expected.
func TestParseLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr string
	}{
		{name: "empty uses default", raw: "", want: 20},
		{name: "lower bound", raw: "1", want: 1},
		{name: "upper clamp", raw: "120", want: 100},
		{name: "invalid", raw: "nope", wantErr: "limit must be an integer"},
		{name: "zero", raw: "0", wantErr: "limit must be greater than zero"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseLimit(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseLimit(%q) error = nil, want %q", tt.raw, tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("parseLimit(%q) error = %q, want %q", tt.raw, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLimit(%q) error = %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("parseLimit(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

// TestParseOptionalTime verifies the first non-empty RFC3339 value is parsed and normalized to UTC.
func TestParseOptionalTime(t *testing.T) {
	t.Parallel()

	ts := "2025-01-02T03:04:05+09:00"
	got, err := parseOptionalTime("", "  ", ts)
	if err != nil {
		t.Fatalf("parseOptionalTime() error = %v", err)
	}
	if got == nil {
		t.Fatalf("parseOptionalTime() = nil, want timestamp")
	}
	want := time.Date(2025, 1, 1, 18, 4, 5, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("parseOptionalTime() = %v, want %v", got, want)
	}
}

// TestBuildEventQuery verifies request query parameters are normalized into an event query.
func TestBuildEventQuery(t *testing.T) {
	t.Parallel()

	query := url.Values{}
	query.Set("connector", "GitHub")
	query.Set("source_uri", "/src")
	query.Set("q", "  foo  ")
	query.Set("limit", "42")
	query.Set("since", "2025-01-01T00:00:00Z")
	query.Set("until", "2025-01-02T00:00:00Z")
	req := httptest.NewRequest("GET", "/artifacts?"+query.Encode(), nil)
	got, err := buildEventQuery(req)
	if err != nil {
		t.Fatalf("buildEventQuery() error = %v", err)
	}

	wantSince, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	wantUntil, _ := time.Parse(time.RFC3339, "2025-01-02T00:00:00Z")
	want := repository.EventQuery{
		Connector: "github",
		SourceURI: "/src",
		Text:      "foo",
		Since:     &wantSince,
		Until:     &wantUntil,
		Limit:     42,
	}

	if got.Connector != want.Connector || got.SourceURI != want.SourceURI || got.Text != want.Text || got.Limit != want.Limit {
		t.Errorf("buildEventQuery() = %#v, want %#v", got, want)
	}
	if got.Since == nil || !got.Since.Equal(*want.Since) {
		t.Errorf("buildEventQuery() Since = %v, want %v", got.Since, want.Since)
	}
	if got.Until == nil || !got.Until.Equal(*want.Until) {
		t.Errorf("buildEventQuery() Until = %v, want %v", got.Until, want.Until)
	}
}

// TestResolveWorkspace verifies workspaces are resolved by path or by fallback list lookup.
func TestResolveWorkspace(t *testing.T) {
	t.Parallel()

	ws := repository.Workspace{ID: "ws-1", Path: "/tmp/context-os"}
	tests := []struct {
		name      string
		repo      fakeWorkspaceRepo
		ref       string
		wantID    string
		wantErrIs error
	}{
		{
			name: "get by path",
			repo: fakeWorkspaceRepo{
				getByPath: map[string]*repository.Workspace{"/tmp/context-os": &ws},
			},
			ref:    "/tmp/context-os",
			wantID: "ws-1",
		},
		{
			name: "fallback list lookup",
			repo: fakeWorkspaceRepo{
				workspaces: []repository.Workspace{ws},
			},
			ref:    "ws-1",
			wantID: "ws-1",
		},
		{
			name:      "not found",
			repo:      fakeWorkspaceRepo{},
			ref:       "missing",
			wantErrIs: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &Handler{workspaces: tt.repo, events: fakeEventRepo{}}
			got, err := h.resolveWorkspace(context.Background(), tt.ref)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("resolveWorkspace() error = %v, want %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveWorkspace() error = %v", err)
			}
			if got.ID != tt.wantID {
				t.Errorf("resolveWorkspace() ID = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

// TestBuildEventQueryIgnoresWhitespace verifies the parser trims whitespace-heavy input values.
func TestBuildEventQueryIgnoresWhitespace(t *testing.T) {
	t.Parallel()

	query := url.Values{}
	query.Set("connector", "  slack  ")
	query.Set("source_uri", "  /rooms/123  ")
	query.Set("q", "  hello world  ")
	req := httptest.NewRequest("GET", "/artifacts?"+query.Encode(), nil)
	got, err := buildEventQuery(req)
	if err != nil {
		t.Fatalf("buildEventQuery() error = %v", err)
	}
	want := repository.EventQuery{
		Connector: "slack",
		SourceURI: "/rooms/123",
		Text:      "hello world",
		Limit:     20,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildEventQuery() = %#v, want %#v", got, want)
	}
}

// TestNoisyLiveEvidenceIDsMatchesOldNoisyRows verifies cleanup targets old live evidence without matching normal source artifacts.
func TestNoisyLiveEvidenceIDsMatchesOldNoisyRows(t *testing.T) {
	t.Parallel()

	events := []repository.IngestEvent{
		{
			ID:        "path-fragment",
			Connector: "googledrive",
			SourceURI: "drive.google.com/file",
			Metadata:  map[string]string{"evidence_kind": "live_chat_answer"},
		},
		{
			ID:        "duplicate-full-answer",
			Connector: "jira",
			SourceURI: "BKGDEV-8096",
			Body:      "Full answer saved for every source.",
			Metadata: map[string]string{
				"evidence_kind":   "live_chat_answer",
				"related_sources": "jira:BKGDEV-8096,jira:BKGDEV-8466",
			},
		},
		{
			ID:        "clean-section",
			Connector: "googledrive",
			SourceURI: "https://docs.google.com/spreadsheets/d/abc/edit",
			Body:      "Source: Google Drive · mapping.xlsx\nSummary: Mapping.",
			Metadata: map[string]string{
				"evidence_kind": "live_chat_answer",
				"source_label":  "Google Drive · mapping.xlsx",
			},
		},
		{
			ID:        "normal-source",
			Connector: "github",
			SourceURI: "sx-tane/context-os",
			Metadata:  map[string]string{},
		},
	}

	got := noisyLiveEvidenceIDs(events)

	want := []string{"path-fragment", "duplicate-full-answer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("noisyLiveEvidenceIDs() = %#v, want %#v", got, want)
	}
}

// TestCleanupLiveEvidenceDeletesSelectedRows verifies the cleanup endpoint deletes only selected noisy live evidence.
func TestCleanupLiveEvidenceDeletesSelectedRows(t *testing.T) {
	t.Parallel()

	ws := repository.Workspace{ID: "ws-1", Path: "/workspace"}
	events := &cleanupEventRepo{events: []repository.IngestEvent{{
		ID:        "enum",
		Connector: "googledrive",
		SourceURI: "enum/value",
		Metadata:  map[string]string{"evidence_kind": "live_chat_answer"},
	}}}
	graph := &fakeGraphEvidenceDeleter{}
	handler := NewHandler(fakeWorkspaceRepo{
		getByPath: map[string]*repository.Workspace{"/workspace": &ws},
	}, events).WithGraphEvidenceDeleter(graph)
	req := httptest.NewRequest("POST", "/artifacts/live-evidence/cleanup?workspace_id=/workspace", nil)
	res := httptest.NewRecorder()

	handler.CleanupLiveEvidence(res, req)

	if res.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if !reflect.DeepEqual(events.deleted, []string{"enum"}) {
		t.Fatalf("deleted = %#v, want enum row", events.deleted)
	}
	if !reflect.DeepEqual(graph.deleted, []string{"enum"}) {
		t.Fatalf("graph deleted = %#v, want enum row", graph.deleted)
	}
}

// TestDeleteDeletesExplicitArtifactIDs verifies the delete endpoint removes user-selected workspace artifacts by ID.
func TestDeleteDeletesExplicitArtifactIDs(t *testing.T) {
	t.Parallel()

	ws := repository.Workspace{ID: "ws-1", Path: "/workspace"}
	events := &cleanupEventRepo{}
	graph := &fakeGraphEvidenceDeleter{result: repository.GraphCleanupResult{
		DeletedEntityCount:       1,
		DeletedRelationshipCount: 2,
	}}
	handler := NewHandler(fakeWorkspaceRepo{
		getByPath: map[string]*repository.Workspace{"/workspace": &ws},
	}, events).WithGraphEvidenceDeleter(graph)
	req := httptest.NewRequest("POST", "/artifacts/delete", strings.NewReader(`{"workspace_id":"/workspace","ids":[" evt-a ","evt-b","evt-a",""]}`))
	res := httptest.NewRecorder()

	handler.Delete(res, req)

	if res.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if !reflect.DeepEqual(events.deleted, []string{"evt-a", "evt-b"}) {
		t.Fatalf("deleted = %#v, want cleaned selected IDs", events.deleted)
	}
	if !reflect.DeepEqual(graph.deleted, []string{"evt-a", "evt-b"}) {
		t.Fatalf("graph deleted = %#v, want cleaned selected IDs", graph.deleted)
	}
	var body CleanupResult
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.DeletedCount != 2 || body.MatchedCount != 2 {
		t.Fatalf("body = %#v, want two matched and deleted IDs", body)
	}
	if body.DeletedGraphEntityCount != 1 || body.DeletedGraphRelationCount != 2 {
		t.Fatalf("body graph counts = %#v, want deleted graph counts", body)
	}
}
