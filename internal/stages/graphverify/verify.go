// Package graphverify validates cross-source graph relationships from local evidence.
package graphverify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
	"context-os/internal/stages/relationship"
)

const (
	// MinConfidence is the minimum accepted confidence for verifier-proposed edges.
	MinConfidence = 0.75

	// MetadataProvider records which verifier proposed an accepted edge.
	MetadataProvider = "graph_verifier_provider"
	// MetadataTraceID records the verification trace for accepted edges.
	MetadataTraceID = "graph_verifier_trace_id"
	// MetadataEvidenceIDs records the local evidence IDs that support an edge.
	MetadataEvidenceIDs = "graph_verifier_evidence_ids"
)

// Assistant proposes cross-source relationships from a local evidence snapshot.
type Assistant interface {
	// Verify proposes relationships from existing Local DB evidence.
	Verify(ctx context.Context, snapshot Snapshot) ([]types.Relationship, error)
	// Provider returns a provenance label for accepted relationships.
	Provider() string
}

// Service verifies and persists cross-source relationships for one workspace.
type Service struct {
	Events    repository.EventRepository
	Entities  repository.EntityRepository
	Assistant Assistant
	Limit     int
}

// Snapshot is the bounded local evidence sent to a graph verifier.
type Snapshot struct {
	WorkspaceID   string
	TraceID       string
	Events        []repository.IngestEvent
	Entities      []entities.CanonicalEntity
	Relationships []types.Relationship
}

// Result reports the verifier outcome.
type Result struct {
	AcceptedRelationshipCount int
	Provider                  string
	TraceID                   string
}

// VerifyWorkspace asks the configured assistant for cross-source relationships and persists accepted edges.
func (s Service) VerifyWorkspace(ctx context.Context, workspaceID string) (Result, error) {
	if s.Events == nil || s.Entities == nil || s.Assistant == nil {
		return Result{}, nil
	}
	limit := s.Limit
	if limit <= 0 {
		limit = 80
	}
	events, err := s.Events.Query(ctx, workspaceID, repository.EventQuery{Limit: limit})
	if err != nil {
		return Result{}, fmt.Errorf("graph verifier: query events: %w", err)
	}
	canonical, err := s.Entities.ListEntities(ctx, workspaceID, "")
	if err != nil {
		return Result{}, fmt.Errorf("graph verifier: list entities: %w", err)
	}
	if len(events) == 0 || len(canonical) < 2 {
		return Result{}, nil
	}
	existing, err := s.Entities.ListRelationships(ctx, workspaceID, nil)
	if err != nil {
		return Result{}, fmt.Errorf("graph verifier: list relationships: %w", err)
	}
	traceID := fmt.Sprintf("graph-verify-%d", time.Now().UTC().UnixNano())
	snapshot := Snapshot{
		WorkspaceID:   workspaceID,
		TraceID:       traceID,
		Events:        events,
		Entities:      canonical,
		Relationships: existing,
	}
	proposed, err := s.Assistant.Verify(ctx, snapshot)
	if err != nil {
		return Result{}, fmt.Errorf("graph verifier: assistant: %w", err)
	}
	accepted := Accept(snapshot, s.Assistant.Provider(), proposed)
	if len(accepted) == 0 {
		return Result{Provider: s.Assistant.Provider(), TraceID: traceID}, nil
	}
	if err := s.Entities.UpsertRelationships(ctx, workspaceID, accepted); err != nil {
		return Result{}, fmt.Errorf("graph verifier: persist relationships: %w", err)
	}
	return Result{
		AcceptedRelationshipCount: len(accepted),
		Provider:                  s.Assistant.Provider(),
		TraceID:                   traceID,
	}, nil
}

// Accept validates proposed cross-source relationships against a local snapshot.
func Accept(snapshot Snapshot, provider string, proposed []types.Relationship) []types.Relationship {
	knownEntities := map[string]entities.CanonicalEntity{}
	for _, entity := range snapshot.Entities {
		knownEntities[entity.Entity.ID] = entity
	}
	knownEvents := map[string]struct{}{}
	for _, event := range snapshot.Events {
		knownEvents[event.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, rel := range snapshot.Relationships {
		seen[rel.ID] = struct{}{}
	}
	accepted := make([]types.Relationship, 0, len(proposed))
	for _, rel := range proposed {
		from, fromOK := knownEntities[rel.FromID]
		to, toOK := knownEntities[rel.ToID]
		if !fromOK || !toOK || rel.FromID == rel.ToID {
			continue
		}
		if from.Entity.SourceID == to.Entity.SourceID {
			continue
		}
		if !knownRelationshipKind(rel.Kind) || rel.Confidence < MinConfidence {
			continue
		}
		evidence := compactEvidence(rel.Evidence, knownEvents)
		if len(evidence) == 0 {
			continue
		}
		rel.ID = relationshipID(rel.FromID, rel.ToID, rel.Kind)
		if _, dup := seen[rel.ID]; dup {
			continue
		}
		if err := relationship.Validate(rel); err != nil {
			continue
		}
		seen[rel.ID] = struct{}{}
		rel.Evidence = evidence
		rel.Metadata = verifierMetadata(rel.Metadata, provider, snapshot.TraceID, evidence)
		accepted = append(accepted, rel)
	}
	return accepted
}

func compactEvidence(values []string, known map[string]struct{}) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := known[value]; !ok {
			continue
		}
		if _, dup := seen[value]; dup {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func verifierMetadata(input map[string]string, provider, traceID string, evidence []string) map[string]string {
	out := make(map[string]string, len(input)+5)
	for key, value := range input {
		out[key] = value
	}
	out[relationship.MetadataAssistive] = "true"
	out[relationship.MetadataAssistProvider] = provider
	out[MetadataProvider] = provider
	out[MetadataTraceID] = traceID
	out[MetadataEvidenceIDs] = strings.Join(evidence, ",")
	return out
}

func knownRelationshipKind(kind types.RelationshipKind) bool {
	switch kind {
	case types.CoOccursInDocument,
		types.RequirementAffectsAPI,
		types.RequirementAffectsService,
		types.APIBackedByDB,
		types.EnumConstrainsField,
		types.ServiceDependsOn:
		return true
	default:
		return false
	}
}

func relationshipID(fromID, toID string, kind types.RelationshipKind) string {
	return fmt.Sprintf("%s->%s:%s", fromID, toID, kind)
}
