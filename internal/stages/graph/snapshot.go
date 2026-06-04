package graph

import (
	"encoding/json" // deterministic JSON serialization of graph snapshots
	"fmt"           // wrap filesystem and decode errors with context
	"os"            // read and write snapshot files on the local filesystem
	"path/filepath" // build snapshot paths in a portable way

	"context-os/domain/entities" // CanonicalEntity persisted in the snapshot
	"context-os/domain/types"    // Relationship persisted in the snapshot
)

// snapshotSchemaVersion identifies the on-disk snapshot format so future changes stay replay-safe.
const snapshotSchemaVersion = "v1"

// graphSnapshot is the persisted, point-in-time view of a context graph, including history.
// It is the local-first persistence format for ContextOS organizational memory.
type graphSnapshot struct {
	SchemaVersion       string                                `json:"schema_version"`       // on-disk format version
	Entities            map[string]entities.CanonicalEntity   `json:"entities"`             // current entity by ID
	Relationships       map[string]types.Relationship         `json:"relationships"`        // current relationship by ID
	EntityHistory       map[string][]entities.CanonicalEntity `json:"entity_history"`       // every recorded entity version by ID
	RelationshipHistory map[string][]types.Relationship       `json:"relationship_history"` // every recorded relationship version by ID
}

// SaveSnapshot writes the graph to dir/<name>.json as deterministic JSON and returns the path.
// The same graph state always produces byte-identical output so snapshots are replay-safe and
// suitable for regression comparison.
func (g *ContextGraph) SaveSnapshot(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create snapshot dir %q: %w", dir, err)
	}
	data, err := json.MarshalIndent(g.toSnapshot(), "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal snapshot: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write snapshot %q: %w", path, err)
	}
	return path, nil
}

// LoadSnapshot reads a snapshot file and reconstructs a ContextGraph with its history intact.
func LoadSnapshot(path string) (*ContextGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %q: %w", path, err)
	}
	var snapshot graphSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode snapshot %q: %w", path, err)
	}
	if snapshot.SchemaVersion != snapshotSchemaVersion {
		return nil, fmt.Errorf("unsupported snapshot schema %q, want %q", snapshot.SchemaVersion, snapshotSchemaVersion)
	}
	return snapshot.toGraph(), nil
}

// toSnapshot copies the graph into its serializable snapshot form.
func (g *ContextGraph) toSnapshot() graphSnapshot {
	return graphSnapshot{
		SchemaVersion:       snapshotSchemaVersion,
		Entities:            g.Entities,
		Relationships:       g.Relationships,
		EntityHistory:       g.EntityHistory,
		RelationshipHistory: g.RelationshipHistory,
	}
}

// toGraph reconstructs a ContextGraph from a decoded snapshot, initialising any nil maps.
func (s graphSnapshot) toGraph() *ContextGraph {
	g := New()
	if s.Entities != nil {
		g.Entities = s.Entities
	}
	if s.Relationships != nil {
		g.Relationships = s.Relationships
	}
	if s.EntityHistory != nil {
		g.EntityHistory = s.EntityHistory
	}
	if s.RelationshipHistory != nil {
		g.RelationshipHistory = s.RelationshipHistory
	}
	return g
}
