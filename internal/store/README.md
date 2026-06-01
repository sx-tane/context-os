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
replaying the same pipeline run is safe. Event dedup uses `(id, workspace_id)`.
Entity and relationship confidence is merged upward (`GREATEST`).

## Usage

```go
sqlDB, _ := db.Open(migrations.Files)
ws := store.NewWorkspaceStore(sqlDB)
ev := store.NewEventStore(sqlDB)
```

Pass store instances into `pipeline.Stores` to persist pipeline output per workspace.
