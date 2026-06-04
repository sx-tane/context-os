// Package store provides PostgreSQL-backed implementations of the domain repository interfaces.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"context-os/domain/entities"
	"context-os/domain/repository"
	"context-os/domain/types"

	"github.com/lib/pq"
)

var workspaceScopedMemoryTables = []string{
	"audit_log",
	"workspace_ui_state",
	"connector_syncs",
	"mismatches",
	"relationships",
	"entities",
	"ingest_events",
}

// ─── Workspace ───────────────────────────────────────────────────────────────

// WorkspaceStore is the PostgreSQL-backed WorkspaceRepository.
type WorkspaceStore struct {
	db *sql.DB
}

// NewWorkspaceStore returns a WorkspaceStore backed by the provided connection pool.
func NewWorkspaceStore(db *sql.DB) *WorkspaceStore {
	return &WorkspaceStore{db: db}
}

// Upsert creates or updates a workspace record identified by its path.
func (s *WorkspaceStore) Upsert(ctx context.Context, w repository.Workspace) (repository.Workspace, error) {
	now := time.Now().UTC()
	if w.ID == "" {
		w.ID = workspaceIDFromPath(w.Path)
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	w.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (path) DO UPDATE SET
			name       = EXCLUDED.name,
			updated_at = EXCLUDED.updated_at
	`, w.ID, w.Name, w.Path, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return repository.Workspace{}, fmt.Errorf("store: upsert workspace: %w", err)
	}
	return w, nil
}

// GetByPath retrieves a workspace by its absolute path. Returns nil, nil when not found.
func (s *WorkspaceStore) GetByPath(ctx context.Context, path string) (*repository.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, path, created_at, updated_at
		FROM workspaces WHERE path = $1
	`, path)
	var w repository.Workspace
	if err := row.Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt, &w.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get workspace by path: %w", err)
	}
	return &w, nil
}

// List returns all registered workspaces ordered by created_at desc.
func (s *WorkspaceStore) List(ctx context.Context) ([]repository.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, path, created_at, updated_at
		FROM workspaces ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list workspaces: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.Workspace
	for rows.Next() {
		var w repository.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan workspace: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// DeleteByPath deletes a workspace by path and removes all workspace-scoped memory rows.
func (s *WorkspaceStore) DeleteByPath(ctx context.Context, path string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin delete workspace by path: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var workspaceID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM workspaces WHERE path = $1
	`, path).Scan(&workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("store: find workspace by path for delete: %w", err)
	}

	for _, table := range workspaceScopedMemoryTables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE workspace_id = $1", table), workspaceID); err != nil {
			return fmt.Errorf("store: delete %s for workspace: %w", table, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM workspaces WHERE id = $1
	`, workspaceID); err != nil {
		return fmt.Errorf("store: delete workspace row by path: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit delete workspace by path: %w", err)
	}
	return nil
}

// ─── Workspace UI State ─────────────────────────────────────────────────────

// WorkspaceUIStateStore is the PostgreSQL-backed WorkspaceUIStateRepository.
type WorkspaceUIStateStore struct {
	db *sql.DB
}

// NewWorkspaceUIStateStore returns a WorkspaceUIStateStore backed by the provided connection pool.
func NewWorkspaceUIStateStore(db *sql.DB) *WorkspaceUIStateStore {
	return &WorkspaceUIStateStore{db: db}
}

// Get returns a workspace UI state document by key.
func (s *WorkspaceUIStateStore) Get(ctx context.Context, workspaceID, stateKey string) (*repository.WorkspaceUIState, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, state_key, payload_json, updated_at
		FROM workspace_ui_state
		WHERE workspace_id = $1 AND state_key = $2
	`, workspaceID, stateKey)
	var state repository.WorkspaceUIState
	if err := row.Scan(&state.WorkspaceID, &state.StateKey, &state.PayloadJSON, &state.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get workspace UI state: %w", err)
	}
	return &state, nil
}

// Put creates or replaces a workspace UI state document by key.
func (s *WorkspaceUIStateStore) Put(ctx context.Context, state repository.WorkspaceUIState) error {
	if len(state.PayloadJSON) == 0 {
		state.PayloadJSON = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_ui_state (workspace_id, state_key, payload_json, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (workspace_id, state_key) DO UPDATE SET
			payload_json = EXCLUDED.payload_json,
			updated_at   = EXCLUDED.updated_at
	`, state.WorkspaceID, state.StateKey, state.PayloadJSON)
	if err != nil {
		return fmt.Errorf("store: put workspace UI state: %w", err)
	}
	return nil
}

// ─── Events ──────────────────────────────────────────────────────────────────

// EventStore is the PostgreSQL-backed EventRepository.
type EventStore struct {
	db *sql.DB
}

// NewEventStore returns an EventStore backed by the provided connection pool.
func NewEventStore(db *sql.DB) *EventStore {
	return &EventStore{db: db}
}

// UpsertBatch writes events, updating any with a duplicate (id, workspace_id).
// Returns the number of rows inserted or updated.
func (s *EventStore) UpsertBatch(ctx context.Context, workspaceID string, evts []repository.IngestEvent) (int, error) {
	if len(evts) == 0 {
		return 0, nil
	}
	var written int
	for _, e := range evts {
		if e.IngestedAt.IsZero() {
			e.IngestedAt = time.Now().UTC()
		}
		metaJSON, err := json.Marshal(e.Metadata)
		if err != nil {
			return written, fmt.Errorf("store: marshal metadata for event %s: %w", e.ID, err)
		}
		r, err := s.db.ExecContext(ctx, `
				INSERT INTO ingest_events
					(id, workspace_id, connector, source_uri, event_type, title, body,
					 content_hash, metadata, schema_version, ingested_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
				ON CONFLICT (id, workspace_id) DO UPDATE SET
					connector      = EXCLUDED.connector,
					source_uri     = EXCLUDED.source_uri,
					event_type     = EXCLUDED.event_type,
					title          = EXCLUDED.title,
					body           = EXCLUDED.body,
					content_hash   = EXCLUDED.content_hash,
					metadata       = EXCLUDED.metadata,
					schema_version = EXCLUDED.schema_version,
					ingested_at    = EXCLUDED.ingested_at
			`,
			e.ID, workspaceID, e.Connector, e.SourceURI, e.EventType,
			e.Title, e.Body, e.ContentHash, metaJSON, e.SchemaVersion, e.IngestedAt,
		)
		if err != nil {
			return written, fmt.Errorf("store: upsert event %s: %w", e.ID, err)
		}
		n, _ := r.RowsAffected()
		written += int(n)
	}
	return written, nil
}

// ListByWorkspace returns events for a workspace ordered by ingested_at desc.
func (s *EventStore) ListByWorkspace(ctx context.Context, workspaceID, connector string, limit int) ([]repository.IngestEvent, error) {
	return s.Query(ctx, workspaceID, repository.EventQuery{
		Connector: connector,
		Limit:     limit,
	})
}

// Query returns events for a workspace ordered by ingested_at desc using optional artifact filters.
func (s *EventStore) Query(ctx context.Context, workspaceID string, eventQuery repository.EventQuery) ([]repository.IngestEvent, error) {
	query := `SELECT id, workspace_id, connector, source_uri, event_type, title, body,
	                 content_hash, metadata, schema_version, ingested_at
	          FROM ingest_events
	          WHERE workspace_id = $1`
	args := []any{workspaceID}

	if eventQuery.Connector != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(eventQuery.Connector)))
		query += fmt.Sprintf(" AND connector = $%d", len(args))
	}
	if eventQuery.SourceURI != "" {
		args = append(args, strings.TrimSpace(eventQuery.SourceURI))
		query += fmt.Sprintf(" AND source_uri = $%d", len(args))
	}
	if eventQuery.Since != nil {
		args = append(args, eventQuery.Since.UTC())
		query += fmt.Sprintf(" AND ingested_at >= $%d", len(args))
	}
	if eventQuery.Until != nil {
		args = append(args, eventQuery.Until.UTC())
		query += fmt.Sprintf(" AND ingested_at < $%d", len(args))
	}
	if eventQuery.Text != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(eventQuery.Text))+"%")
		query += fmt.Sprintf(" AND (LOWER(title) LIKE $%d OR LOWER(body) LIKE $%d OR LOWER(source_uri) LIKE $%d)", len(args), len(args), len(args))
	}
	query += " ORDER BY ingested_at DESC"
	if eventQuery.Limit > 0 {
		args = append(args, eventQuery.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.IngestEvent
	for rows.Next() {
		var e repository.IngestEvent
		var metaJSON []byte
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.Connector, &e.SourceURI,
			&e.EventType, &e.Title, &e.Body, &e.ContentHash,
			&metaJSON, &e.SchemaVersion, &e.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan event: %w", err)
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("store: unmarshal event metadata: %w", err)
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteByIDs removes workspace-scoped ingest events by ID and returns the deleted row count.
func (s *EventStore) DeleteByIDs(ctx context.Context, workspaceID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM ingest_events
		WHERE workspace_id = $1 AND id = ANY($2)
	`, workspaceID, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("store: delete events by id: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// Count returns the total number of events for a workspace and optional connector.
func (s *EventStore) Count(ctx context.Context, workspaceID, connector string) (int, error) {
	query := `SELECT COUNT(*) FROM ingest_events WHERE workspace_id = $1`
	args := []any{workspaceID}
	if connector != "" {
		args = append(args, connector)
		query += fmt.Sprintf(" AND connector = $%d", len(args))
	}
	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count events: %w", err)
	}
	return n, nil
}

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

// CleanupGraphNoise removes low-signal graph rows for a workspace and returns matched/deleted counts.
func (s *EntityStore) CleanupGraphNoise(ctx context.Context, workspaceID string) (repository.GraphCleanupResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: begin graph cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var result repository.GraphCleanupResult
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM relationships
		WHERE workspace_id = $1 AND kind = $2 AND confidence < $3
	`, workspaceID, string(types.CoOccursInDocument), 0.6).Scan(&result.MatchedRelationshipCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count noisy relationships: %w", err)
	}

	deletedRelationships, err := deleteRows(ctx, tx, `
		DELETE FROM relationships
		WHERE workspace_id = $1 AND kind = $2 AND confidence < $3
	`, workspaceID, string(types.CoOccursInDocument), 0.6)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete noisy relationships: %w", err)
	}
	result.DeletedRelationshipCount += deletedRelationships

	if err := tx.QueryRowContext(ctx, noisyEntityCountSQL, workspaceID, 0.6, pq.Array(noisyGraphEntityNames)).Scan(&result.MatchedEntityCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count noisy entities: %w", err)
	}

	deletedEntities, err := deleteRows(ctx, tx, noisyEntityDeleteSQL, workspaceID, 0.6, pq.Array(noisyGraphEntityNames))
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete noisy entities: %w", err)
	}
	result.DeletedEntityCount = deletedEntities

	var danglingCount int
	if err := tx.QueryRowContext(ctx, danglingRelationshipCountSQL, workspaceID).Scan(&danglingCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count dangling relationships: %w", err)
	}
	result.MatchedRelationshipCount += danglingCount

	deletedDangling, err := deleteRows(ctx, tx, danglingRelationshipDeleteSQL, workspaceID)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete dangling relationships: %w", err)
	}
	result.DeletedRelationshipCount += deletedDangling

	if err := tx.Commit(); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: commit graph cleanup: %w", err)
	}
	return result, nil
}

const noisyEntityCountSQL = `
	SELECT COUNT(*) FROM entities
	WHERE workspace_id = $1
	  AND extraction_method = 'regex_token'
	  AND confidence < $2
	  AND (
	    LOWER(TRIM(name)) = ANY($3::text[])
	    OR LENGTH(TRIM(name)) < 3
	  )
`

const noisyEntityDeleteSQL = `
	DELETE FROM entities
	WHERE workspace_id = $1
	  AND extraction_method = 'regex_token'
	  AND confidence < $2
	  AND (
	    LOWER(TRIM(name)) = ANY($3::text[])
	    OR LENGTH(TRIM(name)) < 3
	  )
`

const danglingRelationshipCountSQL = `
	SELECT COUNT(*) FROM relationships r
	WHERE r.workspace_id = $1
	  AND (
	    NOT EXISTS (
	      SELECT 1 FROM entities e
	      WHERE e.workspace_id = r.workspace_id AND e.id = r.from_id
	    )
	    OR NOT EXISTS (
	      SELECT 1 FROM entities e
	      WHERE e.workspace_id = r.workspace_id AND e.id = r.to_id
	    )
	  )
`

const danglingRelationshipDeleteSQL = `
	DELETE FROM relationships r
	WHERE r.workspace_id = $1
	  AND (
	    NOT EXISTS (
	      SELECT 1 FROM entities e
	      WHERE e.workspace_id = r.workspace_id AND e.id = r.from_id
	    )
	    OR NOT EXISTS (
	      SELECT 1 FROM entities e
	      WHERE e.workspace_id = r.workspace_id AND e.id = r.to_id
	    )
	  )
`

func deleteRows(ctx context.Context, tx *sql.Tx, query string, args ...any) (int, error) {
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// ─── Mismatches ───────────────────────────────────────────────────────────────

// MismatchStore is the PostgreSQL-backed MismatchRepository.
type MismatchStore struct {
	db *sql.DB
}

// NewMismatchStore returns a MismatchStore backed by the provided connection pool.
func NewMismatchStore(db *sql.DB) *MismatchStore {
	return &MismatchStore{db: db}
}

// UpsertMismatches persists mismatches, updating on conflict.
func (s *MismatchStore) UpsertMismatches(ctx context.Context, workspaceID string, mismatches []types.Mismatch, traceID string) error {
	for _, m := range mismatches {
		entityIDs := m.EntityIDs
		if entityIDs == nil {
			entityIDs = []string{}
		}
		evidence := m.Evidence
		if evidence == nil {
			evidence = []string{}
		}
		affectedRoles := m.AffectedRoles
		if affectedRoles == nil {
			affectedRoles = []string{}
		}
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO mismatches
				(id, workspace_id, mismatch_type, summary, entity_ids, severity,
				 confidence, impact, evidence, affected_roles, recommended, trace_id, detected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
			ON CONFLICT (id, workspace_id) DO UPDATE SET
				summary        = EXCLUDED.summary,
				confidence     = EXCLUDED.confidence,
				evidence       = EXCLUDED.evidence,
				recommended    = EXCLUDED.recommended,
				trace_id       = EXCLUDED.trace_id,
				detected_at    = NOW()
		`,
			m.ID, workspaceID, m.Type, m.Summary,
			aliasesToPGArray(entityIDs), m.Severity, m.Confidence,
			m.Impact, aliasesToPGArray(evidence),
			aliasesToPGArray(affectedRoles), m.Recommended, traceID,
		)
		if err != nil {
			return fmt.Errorf("store: upsert mismatch %s: %w", m.ID, err)
		}
	}
	return nil
}

// ListByWorkspace returns mismatches ordered by detected_at desc.
func (s *MismatchStore) ListByWorkspace(ctx context.Context, workspaceID, severityMin string, limit int) ([]types.Mismatch, error) {
	query := `SELECT id, mismatch_type, summary, entity_ids, severity,
	                 confidence, impact, evidence, affected_roles, recommended
	          FROM mismatches WHERE workspace_id = $1`
	args := []any{workspaceID}

	if severityMin != "" {
		args = append(args, severityMin)
		query += fmt.Sprintf(" AND severity = $%d", len(args))
	}
	query += " ORDER BY detected_at DESC"
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list mismatches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.Mismatch
	for rows.Next() {
		var m types.Mismatch
		var entityIDsRaw, evidenceRaw, affectedRolesRaw string
		if err := rows.Scan(
			&m.ID, &m.Type, &m.Summary, &entityIDsRaw, &m.Severity,
			&m.Confidence, &m.Impact, &evidenceRaw, &affectedRolesRaw, &m.Recommended,
		); err != nil {
			return nil, fmt.Errorf("store: scan mismatch: %w", err)
		}
		m.EntityIDs = parsePGArray(entityIDsRaw)
		m.Evidence = parsePGArray(evidenceRaw)
		m.AffectedRoles = parsePGArray(affectedRolesRaw)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ─── Sync ─────────────────────────────────────────────────────────────────────

// SyncStore is the PostgreSQL-backed SyncRepository.
type SyncStore struct {
	db *sql.DB
}

// NewSyncStore returns a SyncStore backed by the provided connection pool.
func NewSyncStore(db *sql.DB) *SyncStore {
	return &SyncStore{db: db}
}

// Upsert writes or updates the sync state for one connector+URI in a workspace.
func (s *SyncStore) Upsert(ctx context.Context, sync repository.ConnectorSync) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO connector_syncs
			(workspace_id, connector, source_uri, cursor, last_synced_at, event_count, status, last_error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (workspace_id, connector, source_uri) DO UPDATE SET
			cursor         = EXCLUDED.cursor,
			last_synced_at = EXCLUDED.last_synced_at,
			event_count    = EXCLUDED.event_count,
			status         = EXCLUDED.status,
			last_error     = EXCLUDED.last_error
	`,
		sync.WorkspaceID, sync.Connector, sync.SourceURI,
		sync.Cursor, sync.LastSyncedAt, sync.EventCount, sync.Status, sync.LastError,
	)
	if err != nil {
		return fmt.Errorf("store: upsert connector sync: %w", err)
	}
	return nil
}

// Get returns the sync state for a connector+URI pair, or nil if not found.
func (s *SyncStore) Get(ctx context.Context, workspaceID, connector, sourceURI string) (*repository.ConnectorSync, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, connector, source_uri, cursor, last_synced_at,
		       event_count, status, last_error
		FROM connector_syncs
		WHERE workspace_id = $1 AND connector = $2 AND source_uri = $3
	`, workspaceID, connector, sourceURI)

	var sync repository.ConnectorSync
	if err := row.Scan(
		&sync.WorkspaceID, &sync.Connector, &sync.SourceURI,
		&sync.Cursor, &sync.LastSyncedAt, &sync.EventCount, &sync.Status, &sync.LastError,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get connector sync: %w", err)
	}
	return &sync, nil
}

// ListByWorkspace returns all connector syncs for a workspace.
func (s *SyncStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]repository.ConnectorSync, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT workspace_id, connector, source_uri, cursor, last_synced_at,
		       event_count, status, last_error
		FROM connector_syncs WHERE workspace_id = $1
		ORDER BY last_synced_at DESC NULLS LAST
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list connector syncs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []repository.ConnectorSync
	for rows.Next() {
		var sync repository.ConnectorSync
		if err := rows.Scan(
			&sync.WorkspaceID, &sync.Connector, &sync.SourceURI,
			&sync.Cursor, &sync.LastSyncedAt, &sync.EventCount, &sync.Status, &sync.LastError,
		); err != nil {
			return nil, fmt.Errorf("store: scan connector sync: %w", err)
		}
		out = append(out, sync)
	}
	return out, rows.Err()
}

// ─── Audit ────────────────────────────────────────────────────────────────────

// AuditStore is the PostgreSQL-backed AuditRepository.
type AuditStore struct {
	db *sql.DB
}

// NewAuditStore returns an AuditStore backed by the provided connection pool.
func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Log appends an audit event to the audit_log table.
func (s *AuditStore) Log(ctx context.Context, e repository.AuditEvent) error {
	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("store: marshal audit payload: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_log
			(workspace_id, event_type, actor, connector, source_uri, entity_id, trace_id, payload, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		e.WorkspaceID, e.EventType, e.Actor, e.Connector,
		e.SourceURI, e.EntityID, e.TraceID, payloadJSON,
	)
	if err != nil {
		return fmt.Errorf("store: log audit event: %w", err)
	}
	return nil
}

// CountByWorkspace returns the total audit rows for a workspace.
func (s *AuditStore) CountByWorkspace(ctx context.Context, workspaceID string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_log WHERE workspace_id = $1
	`, workspaceID).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count audit rows: %w", err)
	}
	return n, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// workspaceIDFromPath converts an absolute path into a stable workspace ID
// by replacing path separators with underscores and trimming leading ones.
func workspaceIDFromPath(path string) string {
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.TrimPrefix(id, "_")
	if id == "" {
		return "workspace"
	}
	return id
}

// aliasesToPGArray converts a Go string slice into a PostgreSQL text array literal
// suitable for use in parameterised queries as a $N placeholder value.
func aliasesToPGArray(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
	escaped := make([]string, len(ss))
	for i, s := range ss {
		escaped[i] = strings.ReplaceAll(s, `"`, `\"`)
	}
	return `{"` + strings.Join(escaped, `","`) + `"}`
}

// parsePGArray parses a PostgreSQL text array literal (e.g. {a,b,c}) into a
// Go string slice.  It handles the empty array and quoted elements.
func parsePGArray(raw string) []string {
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimPrefix(p, `"`)
		p = strings.TrimSuffix(p, `"`)
		p = strings.ReplaceAll(p, `\"`, `"`)
		out = append(out, p)
	}
	return out
}
