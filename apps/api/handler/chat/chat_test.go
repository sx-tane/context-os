package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
	internalchat "context-os/internal/chat"
)

// TestQueryStartsAsyncEvidenceSaveForConcreteLiveSource verifies non-stream chat returns a saving status for live source evidence.
func TestQueryStartsAsyncEvidenceSaveForConcreteLiveSource(t *testing.T) {
	saver := &fakeEvidenceSaver{done: make(chan EvidenceSaveInput, 1)}
	handler := NewHandler(internalchat.NewServiceWithLiveAnswerer(
		fakeWorkspaces(),
		&fakeEventRepository{},
		&fakeSyncRepository{},
		&fakeLiveAnswerer{answer: "Live repo answer."},
	)).WithEvidenceSaver(saver)

	req := httptest.NewRequest(http.MethodPost, "/chat/query", strings.NewReader(`{"workspace_id":"/workspace","message":"check sx-tane/context-os"}`))
	res := httptest.NewRecorder()
	handler.Query(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["evidence_save_status"] != evidenceStatusSaving {
		t.Fatalf("evidence_save_status = %v, want %s", body["evidence_save_status"], evidenceStatusSaving)
	}
	select {
	case input := <-saver.done:
		if input.SourceURI != "sx-tane/context-os" {
			t.Fatalf("saved SourceURI = %q, want sx-tane/context-os", input.SourceURI)
		}
		if input.Answer != "Live repo answer." {
			t.Fatalf("saved Answer = %q, want live answer", input.Answer)
		}
	case <-time.After(time.Second):
		t.Fatalf("evidence saver was not called")
	}
}

// TestStreamQueryEmitsAnswerBeforeEvidenceResult verifies live answers stream before local evidence persistence completes.
func TestStreamQueryEmitsAnswerBeforeEvidenceResult(t *testing.T) {
	handler := NewHandler(internalchat.NewServiceWithLiveAnswerer(
		fakeWorkspaces(),
		&fakeEventRepository{},
		&fakeSyncRepository{},
		&fakeLiveAnswerer{answer: "Live repo answer."},
	)).WithEvidenceSaver(&fakeEvidenceSaver{eventCount: 2})

	req := httptest.NewRequest(http.MethodPost, "/chat/query/stream", strings.NewReader(`{"workspace_id":"/workspace","message":"check sx-tane/context-os"}`))
	res := httptest.NewRecorder()
	handler.StreamQuery(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	body := res.Body.String()
	answerIndex := strings.Index(body, "event: answer")
	saveIndex := strings.Index(body, "Saving live answer evidence")
	resultIndex := strings.Index(body, "event: result")
	if answerIndex < 0 {
		t.Fatalf("stream missing answer event: %s", body)
	}
	if saveIndex < 0 {
		t.Fatalf("stream missing save progress: %s", body)
	}
	if resultIndex < 0 {
		t.Fatalf("stream missing result event: %s", body)
	}
	if !(answerIndex < saveIndex && saveIndex < resultIndex) {
		t.Fatalf("stream order answer=%d save=%d result=%d, want answer before save before result", answerIndex, saveIndex, resultIndex)
	}
	if !strings.Contains(body, `"evidence_save_status":"saved"`) {
		t.Fatalf("stream missing saved status: %s", body)
	}
	if !strings.Contains(body, `"evidence_event_count":2`) {
		t.Fatalf("stream missing evidence event count: %s", body)
	}
}

// TestLiveAnswerEventBuildsPersistableEvidence verifies live answers become one local source evidence event without another connector read.
func TestLiveAnswerEventBuildsPersistableEvidence(t *testing.T) {
	event := liveAnswerEvent(EvidenceSaveInput{
		Connector: "jira",
		SourceURI: "BKGDEV-8466",
		Answer:    "BKGDEV-8466 is done.",
		Summary:   "Done",
	}, map[string]string{metadataChatEvidence: "true"})

	if event.Source != "jira" {
		t.Fatalf("Source = %q, want jira", event.Source)
	}
	if event.SourceID != "BKGDEV-8466" {
		t.Fatalf("SourceID = %q, want BKGDEV-8466", event.SourceID)
	}
	if event.Content != "BKGDEV-8466 is done." {
		t.Fatalf("Content = %q, want live answer", event.Content)
	}
	if event.Metadata["evidence_kind"] != "live_chat_answer" {
		t.Fatalf("Metadata[evidence_kind] = %q, want live_chat_answer", event.Metadata["evidence_kind"])
	}
}

// TestStreamQuerySkipsConnectorOnlyEvidenceSave verifies broad connector scopes are not auto-ingested.
func TestStreamQuerySkipsConnectorOnlyEvidenceSave(t *testing.T) {
	handler := NewHandler(internalchat.NewServiceWithLiveAnswerer(
		fakeWorkspaces(),
		&fakeEventRepository{},
		&fakeSyncRepository{syncs: []repository.ConnectorSync{{WorkspaceID: "ws1", Connector: "github", SourceURI: "github", Status: "connected"}}},
		&fakeLiveAnswerer{answer: "Live GitHub answer."},
	)).WithEvidenceSaver(&fakeEvidenceSaver{failOnCall: true})

	req := httptest.NewRequest(http.MethodPost, "/chat/query/stream", strings.NewReader(`{"workspace_id":"/workspace","message":"github"}`))
	res := httptest.NewRecorder()
	handler.StreamQuery(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	body := res.Body.String()
	if strings.Contains(body, "event: answer") {
		t.Fatalf("stream should not emit separate answer event for skipped evidence save: %s", body)
	}
	if !strings.Contains(body, `"evidence_save_status":"skipped"`) {
		t.Fatalf("stream missing skipped status: %s", body)
	}
}

type fakeEvidenceSaver struct {
	eventCount int
	done       chan EvidenceSaveInput
	failOnCall bool
}

func (f *fakeEvidenceSaver) Save(ctx context.Context, input EvidenceSaveInput, progress func(string)) (EvidenceSaveResult, error) {
	if f.failOnCall {
		panic("evidence saver should not be called")
	}
	if progress != nil {
		progress("• Saving live answer evidence to Local DB...")
	}
	if f.done != nil {
		f.done <- input
	}
	return EvidenceSaveResult{EventCount: f.eventCount}, nil
}

type fakeLiveAnswerer struct {
	answer string
}

func (f *fakeLiveAnswerer) Answer(ctx context.Context, query internalchat.LiveQuery) (string, error) {
	return f.answer, nil
}

type fakeWorkspaceRepository struct{}

func fakeWorkspaces() *fakeWorkspaceRepository {
	return &fakeWorkspaceRepository{}
}

func (f *fakeWorkspaceRepository) Upsert(ctx context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	return repository.Workspace{}, nil
}

func (f *fakeWorkspaceRepository) GetByPath(ctx context.Context, path string) (*repository.Workspace, error) {
	return &repository.Workspace{ID: "ws1", Name: "workspace", Path: "/workspace"}, nil
}

func (f *fakeWorkspaceRepository) List(ctx context.Context) ([]repository.Workspace, error) {
	return nil, nil
}

type fakeEventRepository struct{}

func (f *fakeEventRepository) UpsertBatch(ctx context.Context, workspaceID string, events []repository.IngestEvent) (int, error) {
	return 0, nil
}

func (f *fakeEventRepository) ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]repository.IngestEvent, error) {
	return nil, nil
}

func (f *fakeEventRepository) Query(ctx context.Context, workspaceID string, query repository.EventQuery) ([]repository.IngestEvent, error) {
	return nil, nil
}

func (f *fakeEventRepository) Count(ctx context.Context, workspaceID, connector string) (int, error) {
	return 0, nil
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
	if f.syncs != nil {
		return f.syncs, nil
	}
	return []repository.ConnectorSync{{WorkspaceID: workspaceID, Connector: "github", SourceURI: "sx-tane/context-os", Status: "connected"}}, nil
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

var _ EvidenceSaver = (*fakeEvidenceSaver)(nil)
var _ internalchat.LiveAnswerer = (*fakeLiveAnswerer)(nil)
var _ repository.WorkspaceRepository = (*fakeWorkspaceRepository)(nil)
var _ repository.EventRepository = (*fakeEventRepository)(nil)
var _ repository.SyncRepository = (*fakeSyncRepository)(nil)
var _ repository.EntityRepository = (*unusedEntityRepository)(nil)
