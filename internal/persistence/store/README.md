# internal/persistence/store

PostgreSQL-backed implementations of the `domain/repository` interfaces.

## Stores

| Type | Interface | Table(s) |
|------|-----------|---------|
| `WorkspaceStore` | `WorkspaceRepository` | `workspaces` |
| `WorkspaceUIStateStore` | `WorkspaceUIStateRepository` | `workspace_ui_state` |
| `EventStore` | `EventRepository`, `EventDeleter` | `ingest_events` |
| `EntityStore` | `EntityRepository`, `GraphEvidenceDeleter`, `GraphNoiseCleaner` | `entities`, `relationships` |
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

`EntityStore.DeleteGraphEvidenceByEventIDs` is reserved for explicit selected
Activity deletion flows. It deletes relationships whose metadata or evidence
points at the selected event IDs, deletes entities whose `source_id` is one of
those event IDs, and removes relationships touching those same scoped entities.

`EntityStore.CleanupGraphNoise` is reserved for the separate graph cleanup flow.
It permanently removes only backend-classified low-signal graph rows: low-confidence
`co_occurs_in_document` links, low-confidence `regex_token` entities with common
noisy names or very short tokens, and relationships dangling after those entity
deletes. It does not remove source artifacts, findings, chat history, connected
sources, connector syncs, or workspace rows, and it never runs during graph query
or analysis.

`WorkspaceUIStateStore` upserts one JSON document per `(workspace_id, state_key)`.
It is used for durable frontend workflow state such as the analysis basket and
finding action checklist. The store does not validate document fields; typed API
handlers validate and normalize payloads before writing.

## Graph Reads

`EntityStore.ListEntities` and `EntityStore.ListRelationships` provide the graph
API with workspace-scoped nodes and persisted edges. Relationship reads can be
limited to a caller-provided entity ID set when the graph response is filtered.

## Workspace Delete

`WorkspaceStore.DeleteByPath` deletes audit log, workspace UI state, connector sync, mismatch,
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
