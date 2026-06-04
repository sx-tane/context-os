# internal/store

PostgreSQL-backed implementations of the `domain/repository` interfaces.

## Stores

| Type | Interface | Table(s) |
|------|-----------|---------|
| `WorkspaceStore` | `WorkspaceRepository` | `workspaces` |
| `EventStore` | `EventRepository`, `EventDeleter` | `ingest_events` |
| `EntityStore` | `EntityRepository` | `entities`, `relationships` |
| `MismatchStore` | `MismatchRepository` | `mismatches` |
| `SyncStore` | `SyncRepository` | `connector_syncs` |

## Idempotency

All upsert operations use `ON CONFLICT … DO NOTHING` or `DO UPDATE` so that
replaying the same pipeline run is safe. Event dedup uses `(id, workspace_id)`,
and duplicate events update the stored connector/source/content fields.
Entity and relationship confidence is merged upward (`GREATEST`).

`EventStore.DeleteByIDs` is reserved for explicit workspace-scoped cleanup flows
such as removing old noisy live-chat evidence rows. It does not run during normal
artifact queries or pipeline ingestion.

## Graph Reads

`EntityStore.ListEntities` and `EntityStore.ListRelationships` provide the graph
API with workspace-scoped nodes and persisted edges. Relationship reads can be
limited to a caller-provided entity ID set when the graph response is filtered.

## Workspace Delete

`WorkspaceStore.DeleteByPath` deletes audit log, connector sync, mismatch,
relationship, entity, ingest event, and workspace rows in one transaction. The
delete is explicit instead of relying only on foreign-key cascade behavior so
Remove cannot degrade into frontend-only hiding.

## Usage

```go
sqlDB, _ := db.Open(migrations.Files)
ws := store.NewWorkspaceStore(sqlDB)
ev := store.NewEventStore(sqlDB)
```

Pass store instances into `pipeline.Stores` to persist pipeline output per workspace.
