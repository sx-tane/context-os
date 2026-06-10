package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"context-os/domain/entities"
	"context-os/domain/types"
)

// ─── Entities & Relationships ─────────────────────────────────────────────────

// EntityStore is the PostgreSQL-backed EntityRepository.
type EntityStore struct {
	db *sql.DB
}

var noisyGraphEntityNames = []string{
	"and",
	"also",
	"among",
	"source",
	"read",
	"fields",
	"field",
	"type",
	"status",
	"content",
}

// NewEntityStore returns an EntityStore backed by the provided connection pool.
func NewEntityStore(db *sql.DB) *EntityStore {
	return &EntityStore{db: db}
}

// UpsertEntities persists canonical entities, updating confidence and aliases on conflict.
func (s *EntityStore) UpsertEntities(ctx context.Context, workspaceID string, canonical []entities.CanonicalEntity) error {
	for _, ce := range canonical {
		metaJSON, err := json.Marshal(ce.Entity.Metadata)
		if err != nil {
			return fmt.Errorf("store: marshal entity metadata %s: %w", ce.Entity.ID, err)
		}
		aliases := ce.Entity.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO entities
				(id, workspace_id, entity_type, name, raw_mention, source_id,
				 confidence, extraction_method, aliases, needs_human,
				 match_layer, conflict_reason, metadata, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW())
			ON CONFLICT (id, workspace_id) DO UPDATE SET
				confidence        = GREATEST(entities.confidence, EXCLUDED.confidence),
				aliases           = EXCLUDED.aliases,
				needs_human       = EXCLUDED.needs_human,
				match_layer       = EXCLUDED.match_layer,
				conflict_reason   = EXCLUDED.conflict_reason,
				metadata          = EXCLUDED.metadata,
				updated_at        = NOW()
		`,
			ce.Entity.ID, workspaceID, string(ce.Entity.Type), ce.Entity.Name,
			ce.Entity.RawMention, ce.Entity.SourceID,
			ce.Confidence, ce.Entity.ExtractionMethod,
			aliasesToPGArray(aliases),
			ce.NeedsHuman, ce.MatchLayer, ce.ConflictReason, metaJSON,
		)
		if err != nil {
			return fmt.Errorf("store: upsert entity %s: %w", ce.Entity.ID, err)
		}
	}
	return nil
}

// UpsertRelationships persists relationships, updating confidence and evidence on conflict.
func (s *EntityStore) UpsertRelationships(ctx context.Context, workspaceID string, rels []types.Relationship) error {
	for _, rel := range rels {
		metaJSON, err := json.Marshal(rel.Metadata)
		if err != nil {
			return fmt.Errorf("store: marshal relationship metadata %s: %w", rel.ID, err)
		}
		evidence := rel.Evidence
		if evidence == nil {
			evidence = []string{}
		}
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO relationships
				(id, workspace_id, from_id, to_id, kind, confidence, evidence, metadata, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
			ON CONFLICT (id, workspace_id) DO UPDATE SET
				confidence = GREATEST(relationships.confidence, EXCLUDED.confidence),
				evidence   = EXCLUDED.evidence,
				metadata   = EXCLUDED.metadata,
				updated_at = NOW()
		`,
			rel.ID, workspaceID, rel.FromID, rel.ToID,
			string(rel.Kind), rel.Confidence, aliasesToPGArray(evidence), metaJSON,
		)
		if err != nil {
			return fmt.Errorf("store: upsert relationship %s: %w", rel.ID, err)
		}
	}
	return nil
}

// ListEntities returns all entities for a workspace, optionally filtered by entityType.
func (s *EntityStore) ListEntities(ctx context.Context, workspaceID, entityType string) ([]entities.CanonicalEntity, error) {
	query := `SELECT id, entity_type, name, raw_mention, source_id, confidence,
	                 extraction_method, aliases, needs_human, match_layer, conflict_reason, metadata
	          FROM entities WHERE workspace_id = $1`
	args := []any{workspaceID}
	if entityType != "" {
		args = append(args, entityType)
		query += fmt.Sprintf(" AND entity_type = $%d", len(args))
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list entities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []entities.CanonicalEntity
	for rows.Next() {
		var ce entities.CanonicalEntity
		var metaJSON []byte
		var aliasesRaw string
		if err := rows.Scan(
			&ce.Entity.ID, &ce.Entity.Type, &ce.Entity.Name, &ce.Entity.RawMention,
			&ce.Entity.SourceID, &ce.Confidence, &ce.Entity.ExtractionMethod,
			&aliasesRaw, &ce.NeedsHuman, &ce.MatchLayer, &ce.ConflictReason, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("store: scan entity: %w", err)
		}
		ce.Entity.Aliases = parsePGArray(aliasesRaw)
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &ce.Entity.Metadata); err != nil {
				return nil, fmt.Errorf("store: unmarshal entity metadata: %w", err)
			}
		}
		out = append(out, ce)
	}
	return out, rows.Err()
}

// ListRelationships returns persisted relationships for a workspace.
func (s *EntityStore) ListRelationships(ctx context.Context, workspaceID string, entityIDs []string) ([]types.Relationship, error) {
	query := `SELECT id, from_id, to_id, kind, confidence, evidence, metadata
	          FROM relationships WHERE workspace_id = $1`
	args := []any{workspaceID}
	ids := compactStrings(entityIDs)
	if len(ids) > 0 {
		args = append(args, aliasesToPGArray(ids))
		query += fmt.Sprintf(" AND (from_id = ANY($%d::text[]) OR to_id = ANY($%d::text[]))", len(args), len(args))
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.Relationship
	for rows.Next() {
		var rel types.Relationship
		var evidenceRaw string
		var metaJSON []byte
		if err := rows.Scan(
			&rel.ID, &rel.FromID, &rel.ToID, &rel.Kind,
			&rel.Confidence, &evidenceRaw, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("store: scan relationship: %w", err)
		}
		rel.Evidence = parsePGArray(evidenceRaw)
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &rel.Metadata); err != nil {
				return nil, fmt.Errorf("store: unmarshal relationship metadata: %w", err)
			}
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

// CountRelationships returns the total relationship count for a workspace.
func (s *EntityStore) CountRelationships(ctx context.Context, workspaceID string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM relationships WHERE workspace_id = $1
	`, workspaceID).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count relationships: %w", err)
	}
	return n, nil
}

// DeleteGraphEvidenceByEventIDs removes graph rows tied to selected source event IDs.
