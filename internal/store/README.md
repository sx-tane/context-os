# internal/store

PostgreSQL-backed implementations of the `domain/repository` interfaces.

## Stores

| Type | Interface | Table(s) |
|------|-----------|---------|
| `WorkspaceStore` | `WorkspaceRepository` | `workspaces` |
| `EventStore` | `EventRepository` | `ingest_events` |
| `EntityStore` | `EntityRepository` | `entities`, `relationships` |
| `MismatchStore` | `MismatchRepository` | `mismatches` |
| `SyncStore` | `SyncRepository` | `connector_syncs` |

## Idempotency

All upsert operations use `ON CONFLICT … DO NOTHING` or `DO UPDATE` so that
replaying the same pipeline run is safe. Event dedup uses `(id, workspace_id)`,
and duplicate events update the stored connector/source/content fields.
Entity and relationship confidence is merged upward (`GREATEST`).

## Graph Reads

`EntityStore.ListEntities` and `EntityStore.ListRelationships` provide the graph
API with workspace-scoped nodes and persisted edges. Relationship reads can be
limited to a caller-provided entity ID set when the graph response is filtered.

## Usage

```go
sqlDB, _ := db.Open(migrations.Files)
ws := store.NewWorkspaceStore(sqlDB)
ev := store.NewEventStore(sqlDB)
```

Pass store instances into `pipeline.Stores` to persist pipeline output per workspace.
