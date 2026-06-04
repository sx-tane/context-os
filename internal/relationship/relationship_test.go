package relationship_test

import (
	"testing"

	"context-os/domain/entities"
	"context-os/domain/types"
	"context-os/internal/relationship"
)

// canonical wraps an entity for terse test setup.
func canonical(id, source string, kind types.EntityType, name string) entities.CanonicalEntity {
	return entities.CanonicalEntity{Entity: types.Entity{ID: id, SourceID: source, Type: kind, Name: name}}
}

// findKind returns the first relationship of the given kind, or false when none exists.
func findKind(rels []types.Relationship, kind types.RelationshipKind) (types.Relationship, bool) {
	for _, rel := range rels {
		if rel.Kind == kind {
			return rel, true
		}
	}
	return types.Relationship{}, false
}

// TestBuildOnlyLinksEntitiesFromSameSource verifies edges never cross document boundaries.
func TestBuildOnlyLinksEntitiesFromSameSource(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("doc-1:a", "doc-1", types.Requirement, "checkout requirement"),
		canonical("doc-2:b", "doc-2", types.APIField, "amount"),
	}

	got := relationship.Build(input)

	if len(got) != 0 {
		t.Fatalf("Build() length = %d, want 0", len(got))
	}
}

// TestBuildEmitsTypedDeliveryEdges verifies entity-type pairs map to the typed vocabulary.
func TestBuildEmitsTypedDeliveryEdges(t *testing.T) {
	tests := []struct {
		name     string
		from     entities.CanonicalEntity
		to       entities.CanonicalEntity
		wantKind types.RelationshipKind
		wantFrom string
		wantTo   string
	}{
		{
			name:     "requirement affects api",
			from:     canonical("d:req", "d", types.Requirement, "checkout"),
			to:       canonical("d:api", "d", types.APIField, "amount"),
			wantKind: types.RequirementAffectsAPI,
			wantFrom: "d:req",
			wantTo:   "d:api",
		},
		{
			name:     "api backed by db",
			from:     canonical("d:api", "d", types.APIField, "amount"),
			to:       canonical("d:col", "d", types.DBColumn, "amount_cents"),
			wantKind: types.APIBackedByDB,
			wantFrom: "d:api",
			wantTo:   "d:col",
		},
		{
			name:     "service depends on dependency",
			from:     canonical("d:svc", "d", types.Service, "billing"),
			to:       canonical("d:dep", "d", types.Dependency, "stripe"),
			wantKind: types.ServiceDependsOn,
			wantFrom: "d:svc",
			wantTo:   "d:dep",
		},
		{
			name:     "reversed input is oriented consistently",
			from:     canonical("d:api", "d", types.APIField, "amount"),
			to:       canonical("d:req", "d", types.Requirement, "checkout"),
			wantKind: types.RequirementAffectsAPI,
			wantFrom: "d:req",
			wantTo:   "d:api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relationship.Build([]entities.CanonicalEntity{tt.from, tt.to})
			rel, ok := findKind(got, tt.wantKind)
			if !ok {
				t.Fatalf("Build() missing kind %q, got %+v", tt.wantKind, got)
			}
			if rel.FromID != tt.wantFrom {
				t.Errorf("FromID = %q, want %q", rel.FromID, tt.wantFrom)
			}
			if rel.ToID != tt.wantTo {
				t.Errorf("ToID = %q, want %q", rel.ToID, tt.wantTo)
			}
			if rel.Confidence <= 0 {
				t.Errorf("Confidence = %v, want > 0", rel.Confidence)
			}
			if len(rel.Evidence) == 0 {
				t.Errorf("Evidence = empty, want source references")
			}
			if rel.Metadata["source_id"] != "d" {
				t.Errorf("Metadata[source_id] = %q, want d", rel.Metadata["source_id"])
			}
		})
	}
}

// TestBuildFallsBackToCoOccurrence verifies untyped pairs still produce an auditable edge.
func TestBuildFallsBackToCoOccurrence(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("d:dep1", "d", types.Dependency, "kafka"),
		canonical("d:dep2", "d", types.Dependency, "redis"),
	}

	got := relationship.Build(input)

	rel, ok := findKind(got, types.CoOccursInDocument)
	if !ok {
		t.Fatalf("Build() missing co-occurrence edge, got %+v", got)
	}
	if rel.Confidence != 0.5 {
		t.Errorf("Confidence = %v, want 0.5", rel.Confidence)
	}
}

// TestBuildSkipsLowConfidenceRegexOnlyCoOccurrence verifies generic regex-only dependency pairs do not create noisy edges.
func TestBuildSkipsLowConfidenceRegexOnlyCoOccurrence(t *testing.T) {
	input := []entities.CanonicalEntity{
		{Entity: types.Entity{ID: "d:dep1", SourceID: "d", Type: types.Dependency, Name: "kafka", ExtractionMethod: "regex_token", Confidence: 0.58}},
		{Entity: types.Entity{ID: "d:dep2", SourceID: "d", Type: types.Dependency, Name: "redis", ExtractionMethod: "regex_token", Confidence: 0.58}},
	}

	got := relationship.Build(input)

	if _, ok := findKind(got, types.CoOccursInDocument); ok {
		t.Fatalf("Build() emitted low-confidence regex co-occurrence edge: %+v", got)
	}
}

// TestBuildCapsGenericCoOccurrenceEdges verifies large same-document inputs do
// not explode into dense generic all-pairs graphs.
func TestBuildCapsGenericCoOccurrenceEdges(t *testing.T) {
	input := make([]entities.CanonicalEntity, 0, 100)
	for i := 0; i < 100; i++ {
		input = append(input, canonical(
			"d:dep"+string(rune('a'+(i%26)))+string(rune('a'+(i/26))),
			"d",
			types.Dependency,
			"dependency",
		))
	}

	got := relationship.Build(input)

	var generic int
	for _, rel := range got {
		if rel.Kind == types.CoOccursInDocument {
			generic++
		}
	}
	if generic > 25 {
		t.Fatalf("co-occurrence edge count = %d, want <= 25", generic)
	}
}

// TestBuildDeterministicIDIncludesKind verifies distinct edge kinds never collide on ID.
func TestBuildDeterministicIDIncludesKind(t *testing.T) {
	input := []entities.CanonicalEntity{
		canonical("d:req", "d", types.Requirement, "checkout"),
		canonical("d:api", "d", types.APIField, "amount"),
	}

	got := relationship.Build(input)

	rel, ok := findKind(got, types.RequirementAffectsAPI)
	if !ok {
		t.Fatalf("Build() missing typed edge, got %+v", got)
	}
	want := "d:req->d:api:" + string(types.RequirementAffectsAPI)
	if rel.ID != want {
		t.Errorf("ID = %q, want %q", rel.ID, want)
	}
}

// TestValidateRejectsInvalidEdges verifies structurally invalid edges are reported.
func TestValidateRejectsInvalidEdges(t *testing.T) {
	tests := []struct {
		name string
		rel  types.Relationship
	}{
		{name: "empty from", rel: types.Relationship{ID: "x", ToID: "b", Kind: types.APIBackedByDB}},
		{name: "empty to", rel: types.Relationship{ID: "x", FromID: "a", Kind: types.APIBackedByDB}},
		{name: "self loop", rel: types.Relationship{ID: "x", FromID: "a", ToID: "a", Kind: types.APIBackedByDB}},
		{name: "empty kind", rel: types.Relationship{ID: "x", FromID: "a", ToID: "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := relationship.Validate(tt.rel); err == nil {
				t.Errorf("Validate() error = nil, want error")
			}
		})
	}

	valid := types.Relationship{ID: "x", FromID: "a", ToID: "b", Kind: types.APIBackedByDB}
	if err := relationship.Validate(valid); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}
