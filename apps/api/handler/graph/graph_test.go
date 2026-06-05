package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	graphhandler "context-os/apps/api/handler/graph"
	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
)

// TestQueryReturnsFrontendGraphShape verifies the graph endpoint returns flat entity fields used by the UI.
func TestQueryReturnsFrontendGraphShape(t *testing.T) {
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		entityRepo{
			entities: []entities.CanonicalEntity{{
				Entity: types.Entity{
					ID:         "entity-1",
					Type:       types.Requirement,
					Name:       "Refund status",
					RawMention: "refundStatus",
					SourceID:   "github://repo/pull/1",
					Aliases:    []string{"refund_status"},
					Metadata:   map[string]string{"connector": "github"},
				},
				Confidence: 0.91,
				MatchLayer: "exact",
				Evidence:   []string{"github://repo/pull/1"},
				Candidates: []entities.MergeCandidate{{
					Alias:      "refund_status",
					Layer:      "exact",
					Confidence: 0.91,
					Evidence:   "canonical key match",
					Accepted:   true,
				}},
			}, {
				Entity: types.Entity{
					ID:       "entity-2",
					Type:     types.APIField,
					Name:     "refundStatus",
					SourceID: "github://repo/pull/1",
				},
				Confidence: 0.86,
			}},
			relationships: []types.Relationship{{
				ID:         "entity-1->entity-2:requirement_affects_api",
				FromID:     "entity-1",
				ToID:       "entity-2",
				Kind:       types.RequirementAffectsAPI,
				Confidence: 0.8,
				Evidence:   []string{"github://repo/pull/1#refundStatus"},
			}},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/graph?workspace_id=/workspace", nil)
	rec := httptest.NewRecorder()

	handler.Query(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Query() status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeObject(t, rec.Body.Bytes())
	if body["count"] != float64(2) {
		t.Fatalf("count = %v, want 2", body["count"])
	}
	if body["entity_count"] != float64(2) {
		t.Fatalf("entity_count = %v, want 2", body["entity_count"])
	}
	if body["relationship_count"] != float64(1) {
		t.Fatalf("relationship_count = %v, want 1", body["relationship_count"])
	}
	entities := objectSlice(t, body, "entities")
	if len(entities) != 2 {
		t.Fatalf("len(entities) = %d, want 2", len(entities))
	}
	entity := entities[0]
	if _, ok := entity["entity"]; ok {
		t.Fatalf("entities[0][entity] present, want flat GraphEntity shape")
	}
	if entity["id"] != "entity-1" {
		t.Fatalf("entities[0][id] = %v, want entity-1", entity["id"])
	}
	if entity["name"] != "Refund status" {
		t.Fatalf("entities[0][name] = %v, want Refund status", entity["name"])
	}
	if entity["type"] != "requirement" {
		t.Fatalf("entities[0][type] = %v, want requirement", entity["type"])
	}
	relationships := objectSlice(t, body, "relationships")
	if len(relationships) != 1 {
		t.Fatalf("len(relationships) = %d, want 1", len(relationships))
	}
	relationship := relationships[0]
	if relationship["from_id"] != "entity-1" {
		t.Fatalf("relationships[0][from_id] = %v, want entity-1", relationship["from_id"])
	}
	if relationship["to_id"] != "entity-2" {
		t.Fatalf("relationships[0][to_id] = %v, want entity-2", relationship["to_id"])
	}
	if relationship["kind"] != "requirement_affects_api" {
		t.Fatalf("relationships[0][kind] = %v, want requirement_affects_api", relationship["kind"])
	}
}

// TestQueryFiltersNoiseByDefaultAndIncludesWithFlag verifies persisted noisy graph rows are hidden unless debugging is requested.
func TestQueryFiltersNoiseByDefaultAndIncludesWithFlag(t *testing.T) {
	repo := entityRepo{
		entities: []entities.CanonicalEntity{{
			Entity: types.Entity{
				ID:         "entity-signal",
				Type:       types.APIField,
				Name:       "travelFeeCommissionTargetFlag",
				SourceID:   "jira://BKGDEV-8466",
				Confidence: 0.82,
				Metadata:   map[string]string{"extraction_method": "codex_label"},
			},
			Confidence: 0.82,
		}, {
			Entity: types.Entity{
				ID:         "entity-noise",
				Type:       types.Dependency,
				Name:       "Source",
				SourceID:   "jira://BKGDEV-8466",
				Confidence: 0.5,
				Metadata:   map[string]string{"extraction_method": "regex_token"},
			},
			Confidence: 0.5,
		}},
		relationships: []types.Relationship{{
			ID:         "entity-signal->entity-noise:co_occurs_in_document",
			FromID:     "entity-signal",
			ToID:       "entity-noise",
			Kind:       types.CoOccursInDocument,
			Confidence: 0.5,
		}},
	}
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		repo,
	)

	req := httptest.NewRequest(http.MethodGet, "/graph?workspace_id=/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Query(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Query() status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeObject(t, rec.Body.Bytes())
	if body["entity_count"] != float64(1) {
		t.Fatalf("entity_count = %v, want 1", body["entity_count"])
	}
	if body["filtered_entity_count"] != float64(1) {
		t.Fatalf("filtered_entity_count = %v, want 1", body["filtered_entity_count"])
	}
	if body["relationship_count"] != float64(0) {
		t.Fatalf("relationship_count = %v, want 0", body["relationship_count"])
	}

	req = httptest.NewRequest(http.MethodGet, "/graph?workspace_id=/workspace&include_noise=true", nil)
	rec = httptest.NewRecorder()
	handler.Query(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Query(include_noise) status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = decodeObject(t, rec.Body.Bytes())
	if body["entity_count"] != float64(2) {
		t.Fatalf("include_noise entity_count = %v, want 2", body["entity_count"])
	}
	if body["relationship_count"] != float64(1) {
		t.Fatalf("include_noise relationship_count = %v, want 1", body["relationship_count"])
	}
}

// TestCleanupRejectsNonPostAndMissingWorkspace verifies graph cleanup requires POST and a workspace reference.
func TestCleanupRejectsNonPostAndMissingWorkspace(t *testing.T) {
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		&cleanupEntityRepo{},
	)

	req := httptest.NewRequest(http.MethodGet, "/graph/cleanup?workspace_id=/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Cleanup(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Cleanup(GET) status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodPost, "/graph/cleanup", nil)
	rec = httptest.NewRecorder()
	handler.Cleanup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Cleanup(missing workspace) status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestCleanupReturnsUnavailableWhenRepoCannotClean verifies cleanup fails explicitly when the entity store lacks delete support.
func TestCleanupReturnsUnavailableWhenRepoCannotClean(t *testing.T) {
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		entityRepo{},
	)

	req := httptest.NewRequest(http.MethodPost, "/graph/cleanup?workspace_id=/workspace", nil)
	rec := httptest.NewRecorder()
	handler.Cleanup(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Cleanup() status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestCleanupDeletesOnlyBackendClassifiedNoise verifies graph cleanup preserves typed and high-confidence rows.
func TestCleanupDeletesOnlyBackendClassifiedNoise(t *testing.T) {
	repo := &cleanupEntityRepo{
		entities: []entities.CanonicalEntity{{
			Entity: types.Entity{
				ID:               "entity-signal",
				Type:             types.APIField,
				Name:             "travelFeeCommissionTargetFlag",
				SourceID:         "jira://BKGDEV-8466",
				ExtractionMethod: "codex_label",
				Confidence:       0.82,
				Metadata:         map[string]string{"extraction_method": "codex_label"},
			},
			Confidence: 0.82,
		}, {
			Entity: types.Entity{
				ID:               "entity-noise",
				Type:             types.Dependency,
				Name:             "Source",
				SourceID:         "jira://BKGDEV-8466",
				ExtractionMethod: "regex_token",
				Confidence:       0.5,
				Metadata:         map[string]string{"extraction_method": "regex_token"},
			},
			Confidence: 0.5,
		}, {
			Entity: types.Entity{
				ID:               "entity-short",
				Type:             types.Dependency,
				Name:             "x",
				SourceID:         "jira://BKGDEV-8466",
				ExtractionMethod: "regex_token",
				Confidence:       0.4,
				Metadata:         map[string]string{"extraction_method": "regex_token"},
			},
			Confidence: 0.4,
		}, {
			Entity: types.Entity{
				ID:               "entity-preserved-regex",
				Type:             types.Service,
				Name:             "Refund Service",
				SourceID:         "jira://BKGDEV-8466",
				ExtractionMethod: "regex_token",
				Confidence:       0.9,
				Metadata:         map[string]string{"extraction_method": "regex_token"},
			},
			Confidence: 0.9,
		}},
		relationships: []types.Relationship{{
			ID:         "entity-signal->entity-noise:co_occurs_in_document",
			FromID:     "entity-signal",
			ToID:       "entity-noise",
			Kind:       types.CoOccursInDocument,
			Confidence: 0.5,
		}, {
			ID:         "entity-signal->entity-short:requirement_affects_api",
			FromID:     "entity-signal",
			ToID:       "entity-short",
			Kind:       types.RequirementAffectsAPI,
			Confidence: 0.8,
		}, {
			ID:         "entity-signal->entity-preserved-regex:requirement_affects_api",
			FromID:     "entity-signal",
			ToID:       "entity-preserved-regex",
			Kind:       types.RequirementAffectsAPI,
			Confidence: 0.8,
		}},
	}
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		repo,
	)

	req := httptest.NewRequest(http.MethodGet, "/graph?workspace_id=/workspace&include_noise=true", nil)
	rec := httptest.NewRecorder()
	handler.Query(rec, req)
	body := decodeObject(t, rec.Body.Bytes())
	if body["entity_count"] != float64(4) {
		t.Fatalf("before cleanup entity_count = %v, want 4", body["entity_count"])
	}
	if body["relationship_count"] != float64(3) {
		t.Fatalf("before cleanup relationship_count = %v, want 3", body["relationship_count"])
	}

	req = httptest.NewRequest(http.MethodPost, "/graph/cleanup?workspace_id=/workspace", nil)
	rec = httptest.NewRecorder()
	handler.Cleanup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Cleanup() status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = decodeObject(t, rec.Body.Bytes())
	if body["matched_entity_count"] != float64(2) {
		t.Fatalf("matched_entity_count = %v, want 2", body["matched_entity_count"])
	}
	if body["deleted_entity_count"] != float64(2) {
		t.Fatalf("deleted_entity_count = %v, want 2", body["deleted_entity_count"])
	}
	if body["matched_relationship_count"] != float64(2) {
		t.Fatalf("matched_relationship_count = %v, want 2", body["matched_relationship_count"])
	}
	if body["deleted_relationship_count"] != float64(2) {
		t.Fatalf("deleted_relationship_count = %v, want 2", body["deleted_relationship_count"])
	}

	req = httptest.NewRequest(http.MethodGet, "/graph?workspace_id=/workspace&include_noise=true", nil)
	rec = httptest.NewRecorder()
	handler.Query(rec, req)
	body = decodeObject(t, rec.Body.Bytes())
	if body["entity_count"] != float64(2) {
		t.Fatalf("after cleanup entity_count = %v, want 2", body["entity_count"])
	}
	if body["relationship_count"] != float64(1) {
		t.Fatalf("after cleanup relationship_count = %v, want 1", body["relationship_count"])
	}
}

// TestDeleteEntityRejectsInvalidRequests verifies manual graph entity deletion validates method, workspace, and entity ID.
func TestDeleteEntityRejectsInvalidRequests(t *testing.T) {
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		&cleanupEntityRepo{},
	)

	req := httptest.NewRequest(http.MethodPost, "/graph/entity?workspace_id=/workspace&entity_id=entity-1", nil)
	rec := httptest.NewRecorder()
	handler.DeleteEntity(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DeleteEntity(POST) status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodDelete, "/graph/entity?entity_id=entity-1", nil)
	rec = httptest.NewRecorder()
	handler.DeleteEntity(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DeleteEntity(missing workspace) status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	req = httptest.NewRequest(http.MethodDelete, "/graph/entity?workspace_id=/workspace", nil)
	rec = httptest.NewRecorder()
	handler.DeleteEntity(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DeleteEntity(missing entity) status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestDeleteEntityRemovesSelectedEntityAndRelationships verifies manual graph entity deletion keeps the graph consistent.
func TestDeleteEntityRemovesSelectedEntityAndRelationships(t *testing.T) {
	repo := &cleanupEntityRepo{
		entities: []entities.CanonicalEntity{{
			Entity: types.Entity{ID: "service-1", Type: types.Service, Name: "PaymentService"},
		}, {
			Entity: types.Entity{ID: "dependency-1", Type: types.Dependency, Name: "OrderIdRepository"},
		}, {
			Entity: types.Entity{ID: "dependency-2", Type: types.Dependency, Name: "TransactionHistoryRepository"},
		}},
		relationships: []types.Relationship{{
			ID:     "service-1->dependency-1",
			FromID: "service-1",
			ToID:   "dependency-1",
			Kind:   types.ServiceDependsOn,
		}, {
			ID:     "dependency-2->service-1",
			FromID: "dependency-2",
			ToID:   "service-1",
			Kind:   types.ServiceDependsOn,
		}},
	}
	handler := graphhandler.NewHandler(
		workspaceRepo{workspace: repository.Workspace{ID: "workspace-1", Path: "/workspace"}},
		repo,
	)

	req := httptest.NewRequest(http.MethodDelete, "/graph/entity?workspace_id=/workspace&entity_id=service-1", nil)
	rec := httptest.NewRecorder()
	handler.DeleteEntity(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DeleteEntity() status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeObject(t, rec.Body.Bytes())
	if body["deleted_entity_count"] != float64(1) {
		t.Fatalf("deleted_entity_count = %v, want 1", body["deleted_entity_count"])
	}
	if body["deleted_relationship_count"] != float64(2) {
		t.Fatalf("deleted_relationship_count = %v, want 2", body["deleted_relationship_count"])
	}
	if len(repo.entities) != 2 {
		t.Fatalf("entities length = %d, want 2", len(repo.entities))
	}
	if len(repo.relationships) != 0 {
		t.Fatalf("relationships length = %d, want 0", len(repo.relationships))
	}
}

type workspaceRepo struct {
	workspace repository.Workspace
}

func (r workspaceRepo) Upsert(_ context.Context, workspace repository.Workspace) (repository.Workspace, error) {
	return workspace, nil
}

func (r workspaceRepo) GetByPath(_ context.Context, path string) (*repository.Workspace, error) {
	if path == r.workspace.Path {
		return &r.workspace, nil
	}
	return nil, nil
}

func (r workspaceRepo) List(_ context.Context) ([]repository.Workspace, error) {
	return []repository.Workspace{r.workspace}, nil
}

type entityRepo struct {
	entities      []entities.CanonicalEntity
	relationships []types.Relationship
}

func (r entityRepo) UpsertEntities(_ context.Context, _ string, _ []entities.CanonicalEntity) error {
	return nil
}

func (r entityRepo) UpsertRelationships(_ context.Context, _ string, _ []types.Relationship) error {
	return nil
}

func (r entityRepo) ListEntities(_ context.Context, _ string, _ string) ([]entities.CanonicalEntity, error) {
	return r.entities, nil
}

func (r entityRepo) ListRelationships(_ context.Context, _ string, _ []string) ([]types.Relationship, error) {
	return r.relationships, nil
}

type cleanupEntityRepo struct {
	entities      []entities.CanonicalEntity
	relationships []types.Relationship
}

func (r *cleanupEntityRepo) UpsertEntities(_ context.Context, _ string, _ []entities.CanonicalEntity) error {
	return nil
}

func (r *cleanupEntityRepo) UpsertRelationships(_ context.Context, _ string, _ []types.Relationship) error {
	return nil
}

func (r *cleanupEntityRepo) ListEntities(_ context.Context, _ string, _ string) ([]entities.CanonicalEntity, error) {
	return append([]entities.CanonicalEntity(nil), r.entities...), nil
}

func (r *cleanupEntityRepo) ListRelationships(_ context.Context, _ string, _ []string) ([]types.Relationship, error) {
	return append([]types.Relationship(nil), r.relationships...), nil
}

func (r *cleanupEntityRepo) CleanupGraphNoise(_ context.Context, _ string) (repository.GraphCleanupResult, error) {
	result := repository.GraphCleanupResult{}
	keptRelationships := make([]types.Relationship, 0, len(r.relationships))
	for _, relationship := range r.relationships {
		if relationship.Kind == types.CoOccursInDocument && relationship.Confidence < 0.6 {
			result.MatchedRelationshipCount++
			result.DeletedRelationshipCount++
			continue
		}
		keptRelationships = append(keptRelationships, relationship)
	}
	r.relationships = keptRelationships

	deletedEntityIDs := map[string]struct{}{}
	keptEntities := make([]entities.CanonicalEntity, 0, len(r.entities))
	for _, entity := range r.entities {
		if isNoisyTestEntity(entity) {
			result.MatchedEntityCount++
			result.DeletedEntityCount++
			deletedEntityIDs[entity.Entity.ID] = struct{}{}
			continue
		}
		keptEntities = append(keptEntities, entity)
	}
	r.entities = keptEntities

	keptRelationships = make([]types.Relationship, 0, len(r.relationships))
	for _, relationship := range r.relationships {
		_, fromDeleted := deletedEntityIDs[relationship.FromID]
		_, toDeleted := deletedEntityIDs[relationship.ToID]
		if fromDeleted || toDeleted {
			result.MatchedRelationshipCount++
			result.DeletedRelationshipCount++
			continue
		}
		keptRelationships = append(keptRelationships, relationship)
	}
	r.relationships = keptRelationships
	return result, nil
}

func (r *cleanupEntityRepo) DeleteGraphEntity(_ context.Context, _ string, entityID string) (repository.GraphCleanupResult, error) {
	result := repository.GraphCleanupResult{}
	keptRelationships := make([]types.Relationship, 0, len(r.relationships))
	for _, relationship := range r.relationships {
		if relationship.FromID == entityID || relationship.ToID == entityID {
			result.MatchedRelationshipCount++
			result.DeletedRelationshipCount++
			continue
		}
		keptRelationships = append(keptRelationships, relationship)
	}
	r.relationships = keptRelationships

	keptEntities := make([]entities.CanonicalEntity, 0, len(r.entities))
	for _, entity := range r.entities {
		if entity.Entity.ID == entityID {
			result.MatchedEntityCount++
			result.DeletedEntityCount++
			continue
		}
		keptEntities = append(keptEntities, entity)
	}
	r.entities = keptEntities
	return result, nil
}

func isNoisyTestEntity(entity entities.CanonicalEntity) bool {
	if entity.Confidence >= 0.6 || entity.Entity.ExtractionMethod != "regex_token" {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(entity.Entity.Name))
	switch name {
	case "and", "also", "among", "source", "read", "fields", "field", "type", "status", "content":
		return true
	default:
		return len(name) < 3
	}
}

func decodeObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return decoded
}

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
