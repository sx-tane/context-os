package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
	internalchat "context-os/internal/runtime/chat"
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
	)).WithEvidenceSaver(&fakeEvidenceSaver{eventCount: 2, graphUpdated: true})

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
	if !strings.Contains(body, `"evidence_graph_status":"updated"`) {
		t.Fatalf("stream missing graph update status: %s", body)
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

// TestEvidenceSaveInputExtractsConcreteSourcesFromBroadLiveAnswer verifies broad connector routes persist concrete provenance found in the answer.
func TestEvidenceSaveInputExtractsConcreteSourcesFromBroadLiveAnswer(t *testing.T) {
	input, ok := evidenceSaveInput(mapChatResult(internalchat.Result{
		WorkspaceID:   "ws1",
		WorkspacePath: "/workspace",
		Connector:     "googledrive",
		SourceURI:     "googledrive",
		Provider:      "codex",
		Answer:        "Spreadsheet: https://docs.google.com/spreadsheets/d/abc/edit. Jira: BKGDEV-8096 and https://acme.atlassian.net/browse/BKGDEV-8466. Slack: #proj-report.",
		Summary:       "Concrete sources",
	}))
	if !ok {
		t.Fatalf("evidenceSaveInput() ok = false, want true")
	}
	want := map[string]bool{
		"googledrive:https://docs.google.com/spreadsheets/d/abc/edit": true,
		"jira:BKGDEV-8096": true,
		"jira:https://acme.atlassian.net/browse/BKGDEV-8466": true,
		"slack:#proj-report": true,
	}
	if len(input.Sources) != len(want) {
		t.Fatalf("Sources length = %d, want %d: %#v", len(input.Sources), len(want), input.Sources)
	}
	for _, source := range input.Sources {
		key := source.Connector + ":" + source.SourceURI
		if !want[key] {
			t.Fatalf("unexpected source %q from %#v", key, input.Sources)
		}
	}
}

// TestEvidenceSaveInputUsesStructuredSectionsBeforeRegex verifies source sections drive evidence persistence ahead of prose extraction.
func TestEvidenceSaveInputUsesStructuredSectionsBeforeRegex(t *testing.T) {
	input, ok := evidenceSaveInput(mapChatResult(internalchat.Result{
		WorkspaceID:   "ws1",
		WorkspacePath: "/workspace",
		Connector:     "googledrive",
		SourceURI:     "googledrive",
		Provider:      "codex",
		Answer:        "Mentions enum/value and https://docs.google.com/spreadsheets/d/noisy/edit in prose.",
		Summary:       "Structured answer",
		AnswerSections: []internalchat.AnswerSection{{
			SourceLabel: "Google Drive · BKGDEV-8096_帳票項目のマッピング確認.xlsx",
			Connector:   "googledrive",
			SourceURI:   "https://docs.google.com/spreadsheets/d/abc/edit",
			Summary:     "Mapping confirmation is pending.",
			Facts:       []string{"Column A maps to field_a."},
		}},
	}))
	if !ok {
		t.Fatalf("evidenceSaveInput() ok = false, want true")
	}
	if len(input.Sources) != 1 {
		t.Fatalf("Sources length = %d, want 1: %#v", len(input.Sources), input.Sources)
	}
	source := input.Sources[0]
	if source.SourceURI != "https://docs.google.com/spreadsheets/d/abc/edit" {
		t.Fatalf("SourceURI = %q, want structured section URI", source.SourceURI)
	}
	if source.Section.SourceLabel != "Google Drive · BKGDEV-8096_帳票項目のマッピング確認.xlsx" {
		t.Fatalf("SourceLabel = %q, want structured label", source.Section.SourceLabel)
	}
}

// TestLiveAnswerEventUsesSectionBody verifies structured source sections save a focused Activity body.
func TestLiveAnswerEventUsesSectionBody(t *testing.T) {
	input := EvidenceSaveInput{
		Connector: "googledrive",
		SourceURI: "https://docs.google.com/spreadsheets/d/abc/edit",
		Answer:    "Full answer with unrelated source.",
		Summary:   "Full summary",
		Sources: []EvidenceSource{{
			Connector: "googledrive",
			SourceURI: "https://docs.google.com/spreadsheets/d/abc/edit",
			Section: response.AnswerSection{
				SourceLabel: "Google Drive · mapping.xlsx",
				Summary:     "Mapping confirmation is pending.",
				Facts:       []string{"Column A maps to field_a."},
			},
		}},
	}
	source := input.Sources[0]
	input.Answer = sourceSectionBody(source, input.Answer)
	input.Summary = sourceSectionSummary(source, input.Summary)

	event := liveAnswerEvent(input, map[string]string{metadataChatEvidence: "true"})

	if event.Subject != "Live chat evidence: Google Drive · mapping.xlsx" {
		t.Fatalf("Subject = %q, want source label title", event.Subject)
	}
	if !strings.Contains(event.Content, "Source: Google Drive · mapping.xlsx") {
		t.Fatalf("Content = %q, want section source", event.Content)
	}
	if strings.Contains(event.Content, "unrelated source") {
		t.Fatalf("Content = %q, want focused section body", event.Content)
	}
	if event.Metadata["source_label"] != "Google Drive · mapping.xlsx" {
		t.Fatalf("Metadata[source_label] = %q, want source label", event.Metadata["source_label"])
	}
}

// TestEvidenceSaveInputSkipsConnectorOnlyAnswerWithoutConcreteSource verifies broad live answers stay read-only without concrete provenance.
func TestEvidenceSaveInputSkipsConnectorOnlyAnswerWithoutConcreteSource(t *testing.T) {
	_, ok := evidenceSaveInput(mapChatResult(internalchat.Result{
		WorkspaceID:   "ws1",
		WorkspacePath: "/workspace",
		Connector:     "googledrive",
		SourceURI:     "googledrive",
		Provider:      "codex",
		Answer:        "There are several recent Drive files, but no visible URLs or concrete source references.",
		Summary:       "Broad answer",
	}))
	if ok {
		t.Fatalf("evidenceSaveInput() ok = true, want false")
	}
}

// TestStreamQueryDoesNotRunLiveLookupAfterEvidenceSave verifies graph/evidence persistence does not trigger another live answer.
func TestStreamQueryDoesNotRunLiveLookupAfterEvidenceSave(t *testing.T) {
	live := &fakeLiveAnswerer{answer: "Live repo answer."}
	handler := NewHandler(internalchat.NewServiceWithLiveAnswerer(
		fakeWorkspaces(),
		&fakeEventRepository{},
		&fakeSyncRepository{},
		live,
	)).WithEvidenceSaver(&fakeEvidenceSaver{eventCount: 1, graphUpdated: true})

	req := httptest.NewRequest(http.MethodPost, "/chat/query/stream", strings.NewReader(`{"workspace_id":"/workspace","message":"check sx-tane/context-os"}`))
	res := httptest.NewRecorder()
	handler.StreamQuery(res, req)

	if live.calls != 1 {
		t.Fatalf("live calls = %d, want 1", live.calls)
	}
	if !strings.Contains(res.Body.String(), `"evidence_save_status":"saved"`) {
		t.Fatalf("stream missing saved status: %s", res.Body.String())
	}
}

type fakeEvidenceSaver struct {
	eventCount   int
	graphUpdated bool
	done         chan EvidenceSaveInput
	failOnCall   bool
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
	return EvidenceSaveResult{EventCount: f.eventCount, GraphUpdated: f.graphUpdated, EntityCount: 3, RelationshipCount: 2}, nil
}

type fakeLiveAnswerer struct {
	answer string
	calls  int
}

func (f *fakeLiveAnswerer) Answer(ctx context.Context, query internalchat.LiveQuery) (string, error) {
	f.calls++
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
