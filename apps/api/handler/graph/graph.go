// Package graph provides the HTTP handler for querying persisted entity graph data.
package graph

import (
	"net/http"
	"strings"

	"context-os/apps/api/response"
	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"
)

// Handler exposes graph query endpoints backed by persistent entity data.
type Handler struct {
	workspaces repository.WorkspaceRepository
	entities   repository.EntityRepository
}

type queryResponse struct {
	WorkspaceID               string              `json:"workspace_id"`
	EntityType                string              `json:"entity_type,omitempty"`
	Count                     int                 `json:"count"`
	EntityCount               int                 `json:"entity_count"`
	RelationshipCount         int                 `json:"relationship_count"`
	FilteredEntityCount       int                 `json:"filtered_entity_count"`
	FilteredRelationshipCount int                 `json:"filtered_relationship_count"`
	TotalEntityCount          int                 `json:"total_entity_count"`
	TotalRelationshipCount    int                 `json:"total_relationship_count"`
	Entities                  []graphEntity       `json:"entities"`
	Relationships             []graphRelationship `json:"relationships"`
}

type graphEntity struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Type           string                `json:"type"`
	Source         string                `json:"source"`
	RawMention     string                `json:"raw_mention,omitempty"`
	Confidence     float64               `json:"confidence"`
	NeedsHuman     bool                  `json:"needs_human,omitempty"`
	MatchLayer     string                `json:"match_layer,omitempty"`
	ConflictReason string                `json:"conflict_reason,omitempty"`
	Evidence       []string              `json:"evidence,omitempty"`
	Aliases        []string              `json:"aliases,omitempty"`
	Candidates     []graphMergeCandidate `json:"candidates,omitempty"`
	Metadata       map[string]string     `json:"metadata,omitempty"`
}

type graphMergeCandidate struct {
	Alias      string  `json:"alias"`
	Layer      string  `json:"layer"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
	Accepted   bool    `json:"accepted"`
}

type graphRelationship struct {
	ID         string            `json:"id"`
	FromID     string            `json:"from_id"`
	ToID       string            `json:"to_id"`
	Kind       string            `json:"kind"`
	Confidence float64           `json:"confidence"`
	Evidence   []string          `json:"evidence,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(workspaces repository.WorkspaceRepository, entities repository.EntityRepository) *Handler {
	return &Handler{workspaces: workspaces, entities: entities}
}

// Query handles GET /graph.
//
// @Summary      Query workspace entity graph
// @Description  Returns persisted canonical entities for a workspace, optionally filtered by entity type.
// @Tags         graph
// @Produce      json
// @Param        workspace_id  query     string  true   "Workspace path or ID"
// @Param        entity_type   query     string  false  "Filter by entity type (e.g. feature, person, service)"
// @Param        include_noise query     bool    false  "Include low-signal regex entities and co-occurrence-only relationships"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /graph [get]
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if workspaceID == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id is required")
		return
	}
	entityType := strings.TrimSpace(r.URL.Query().Get("entity_type"))
	includeNoise := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_noise")), "true")

	// Resolve workspace path to stored ID via WorkspaceRepository.
	ws, err := h.workspaces.GetByPath(r.Context(), workspaceID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	resolvedID := workspaceID
	if ws != nil {
		resolvedID = ws.ID
	}

	canonical, err := h.entities.ListEntities(r.Context(), resolvedID, entityType)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	allGraphEntities := mapEntities(canonical)
	graphEntities := allGraphEntities
	if !includeNoise {
		graphEntities = filterSignalEntities(graphEntities)
	}
	relationships, err := h.entities.ListRelationships(r.Context(), resolvedID, entityIDs(canonical))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	allGraphRelationships := mapRelationships(relationships, allGraphEntities)
	graphRelationships := mapRelationships(relationships, graphEntities)
	if !includeNoise {
		graphRelationships = filterSignalRelationships(graphRelationships)
	}
	response.WriteJSON(w, http.StatusOK, queryResponse{
		WorkspaceID:               resolvedID,
		EntityType:                entityType,
		Count:                     len(graphEntities),
		EntityCount:               len(graphEntities),
		RelationshipCount:         len(graphRelationships),
		FilteredEntityCount:       len(allGraphEntities) - len(graphEntities),
		FilteredRelationshipCount: len(allGraphRelationships) - len(graphRelationships),
		TotalEntityCount:          len(allGraphEntities),
		TotalRelationshipCount:    len(allGraphRelationships),
		Entities:                  graphEntities,
		Relationships:             graphRelationships,
	})
}

func mapEntities(canonical []entities.CanonicalEntity) []graphEntity {
	out := make([]graphEntity, 0, len(canonical))
	for _, ce := range canonical {
		metadata := cloneMetadata(ce.Entity.Metadata)
		if ce.Entity.ExtractionMethod != "" {
			metadata["extraction_method"] = ce.Entity.ExtractionMethod
		}
		candidates := make([]graphMergeCandidate, 0, len(ce.Candidates))
		for _, candidate := range ce.Candidates {
			candidates = append(candidates, graphMergeCandidate{
				Alias:      candidate.Alias,
				Layer:      candidate.Layer,
				Confidence: candidate.Confidence,
				Evidence:   candidate.Evidence,
				Accepted:   candidate.Accepted,
			})
		}
		out = append(out, graphEntity{
			ID:             ce.Entity.ID,
			Name:           ce.Entity.Name,
			Type:           string(ce.Entity.Type),
			Source:         ce.Entity.SourceID,
			RawMention:     ce.Entity.RawMention,
			Confidence:     ce.Confidence,
			NeedsHuman:     ce.NeedsHuman,
			MatchLayer:     ce.MatchLayer,
			ConflictReason: ce.ConflictReason,
			Evidence:       ce.Evidence,
			Aliases:        ce.Entity.Aliases,
			Candidates:     candidates,
			Metadata:       metadata,
		})
	}
	return out
}

func entityIDs(canonical []entities.CanonicalEntity) []string {
	ids := make([]string, 0, len(canonical))
	for _, ce := range canonical {
		if ce.Entity.ID != "" {
			ids = append(ids, ce.Entity.ID)
		}
	}
	return ids
}

func mapRelationships(relationships []types.Relationship, graphEntities []graphEntity) []graphRelationship {
	entitySet := make(map[string]struct{}, len(graphEntities))
	for _, entity := range graphEntities {
		entitySet[entity.ID] = struct{}{}
	}
	out := make([]graphRelationship, 0, len(relationships))
	for _, rel := range relationships {
		if _, ok := entitySet[rel.FromID]; !ok {
			continue
		}
		if _, ok := entitySet[rel.ToID]; !ok {
			continue
		}
		out = append(out, graphRelationship{
			ID:         rel.ID,
			FromID:     rel.FromID,
			ToID:       rel.ToID,
			Kind:       string(rel.Kind),
			Confidence: rel.Confidence,
			Evidence:   rel.Evidence,
			Metadata:   rel.Metadata,
		})
	}
	return out
}

func filterSignalEntities(entities []graphEntity) []graphEntity {
	out := make([]graphEntity, 0, len(entities))
	for _, entity := range entities {
		if isNoisyEntity(entity) {
			continue
		}
		out = append(out, entity)
	}
	return out
}

func filterSignalRelationships(relationships []graphRelationship) []graphRelationship {
	out := make([]graphRelationship, 0, len(relationships))
	for _, relationship := range relationships {
		if relationship.Kind == string(types.CoOccursInDocument) && relationship.Confidence < 0.6 {
			continue
		}
		out = append(out, relationship)
	}
	return out
}

func isNoisyEntity(entity graphEntity) bool {
	method := strings.TrimSpace(entity.Metadata["extraction_method"])
	if method == "" {
		method = strings.TrimSpace(entity.Metadata["method"])
	}
	name := strings.ToLower(strings.TrimSpace(entity.Name))
	if entity.Confidence >= 0.6 {
		return false
	}
	if isCommonGraphLabel(name) {
		return true
	}
	if method != "regex_token" {
		return false
	}
	return name == "" || len(name) < 3
}

func isCommonGraphLabel(name string) bool {
	switch name {
	case "and", "also", "among", "source", "read", "fields", "field", "type", "status", "content":
		return true
	default:
		return false
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
