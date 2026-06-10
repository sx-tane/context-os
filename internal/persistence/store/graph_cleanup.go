package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"context-os/domain/repository"
	"context-os/domain/types"

	"github.com/lib/pq"
)

func (s *EntityStore) DeleteGraphEvidenceByEventIDs(ctx context.Context, workspaceID string, eventIDs []string) (repository.GraphCleanupResult, error) {
	ids := compactStrings(eventIDs)
	if len(ids) == 0 {
		return repository.GraphCleanupResult{}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: begin graph evidence delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var result repository.GraphCleanupResult
	if err := tx.QueryRowContext(ctx, graphEvidenceRelationshipCountSQL, workspaceID, pq.Array(ids)).Scan(&result.MatchedRelationshipCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count graph evidence relationships: %w", err)
	}
	deletedRelationships, err := deleteRows(ctx, tx, graphEvidenceRelationshipDeleteSQL, workspaceID, pq.Array(ids))
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete graph evidence relationships: %w", err)
	}
	result.DeletedRelationshipCount += deletedRelationships

	var entityRelationshipCount int
	if err := tx.QueryRowContext(ctx, graphEvidenceEntityRelationshipCountSQL, workspaceID, pq.Array(ids)).Scan(&entityRelationshipCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count graph evidence entity relationships: %w", err)
	}
	result.MatchedRelationshipCount += entityRelationshipCount
	deletedEntityRelationships, err := deleteRows(ctx, tx, graphEvidenceEntityRelationshipDeleteSQL, workspaceID, pq.Array(ids))
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete graph evidence entity relationships: %w", err)
	}
	result.DeletedRelationshipCount += deletedEntityRelationships

	if err := tx.QueryRowContext(ctx, graphEvidenceEntityCountSQL, workspaceID, pq.Array(ids)).Scan(&result.MatchedEntityCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count graph evidence entities: %w", err)
	}
	deletedEntities, err := deleteRows(ctx, tx, graphEvidenceEntityDeleteSQL, workspaceID, pq.Array(ids))
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete graph evidence entities: %w", err)
	}
	result.DeletedEntityCount = deletedEntities

	if err := tx.Commit(); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: commit graph evidence delete: %w", err)
	}
	return result, nil
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

// DeleteGraphEntity removes one entity and all relationships touching it.
func (s *EntityStore) DeleteGraphEntity(ctx context.Context, workspaceID, entityID string) (repository.GraphCleanupResult, error) {
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return repository.GraphCleanupResult{}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: begin graph entity delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var result repository.GraphCleanupResult
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM entities
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, entityID).Scan(&result.MatchedEntityCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count graph entity: %w", err)
	}

	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM relationships
		WHERE workspace_id = $1 AND (from_id = $2 OR to_id = $2)
	`, workspaceID, entityID).Scan(&result.MatchedRelationshipCount); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: count graph entity relationships: %w", err)
	}

	deletedRelationships, err := deleteRows(ctx, tx, `
		DELETE FROM relationships
		WHERE workspace_id = $1 AND (from_id = $2 OR to_id = $2)
	`, workspaceID, entityID)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete graph entity relationships: %w", err)
	}
	result.DeletedRelationshipCount = deletedRelationships

	deletedEntities, err := deleteRows(ctx, tx, `
		DELETE FROM entities
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, entityID)
	if err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: delete graph entity: %w", err)
	}
	result.DeletedEntityCount = deletedEntities

	if err := tx.Commit(); err != nil {
		return repository.GraphCleanupResult{}, fmt.Errorf("store: commit graph entity delete: %w", err)
	}
	return result, nil
}

const graphEvidenceRelationshipCountSQL = `
	SELECT COUNT(*) FROM relationships
	WHERE workspace_id = $1
	  AND (
	    metadata->>'source_id' = ANY($2::text[])
	    OR string_to_array(COALESCE(metadata->>'graph_verifier_evidence_ids', ''), ',') && $2::text[]
	    OR evidence && $2::text[]
	  )
`

const graphEvidenceRelationshipDeleteSQL = `
	DELETE FROM relationships
	WHERE workspace_id = $1
	  AND (
	    metadata->>'source_id' = ANY($2::text[])
	    OR string_to_array(COALESCE(metadata->>'graph_verifier_evidence_ids', ''), ',') && $2::text[]
	    OR evidence && $2::text[]
	  )
`

const graphEvidenceEntityRelationshipCountSQL = `
	SELECT COUNT(*) FROM relationships r
	WHERE r.workspace_id = $1
	  AND EXISTS (
	    SELECT 1 FROM entities e
	    WHERE e.workspace_id = r.workspace_id
	      AND e.source_id = ANY($2::text[])
	      AND (e.id = r.from_id OR e.id = r.to_id)
	  )
`

const graphEvidenceEntityRelationshipDeleteSQL = `
	DELETE FROM relationships r
	WHERE r.workspace_id = $1
	  AND EXISTS (
	    SELECT 1 FROM entities e
	    WHERE e.workspace_id = r.workspace_id
	      AND e.source_id = ANY($2::text[])
	      AND (e.id = r.from_id OR e.id = r.to_id)
	  )
`

const graphEvidenceEntityCountSQL = `
	SELECT COUNT(*) FROM entities
	WHERE workspace_id = $1 AND source_id = ANY($2::text[])
`

const graphEvidenceEntityDeleteSQL = `
	DELETE FROM entities
	WHERE workspace_id = $1 AND source_id = ANY($2::text[])
`

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
