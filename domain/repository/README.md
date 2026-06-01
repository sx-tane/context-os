# domain/repository

Domain repository interfaces for workspace-scoped persistence.

Implementations live in `internal/store`. Nothing in `domain/` depends on `internal/`.

## Interfaces

| Interface | Purpose |
|-----------|---------|
| `WorkspaceRepository` | Register and retrieve workspace records by path. |
| `EventRepository` | Upsert and query raw ingested source events (idempotent by `id+workspace_id`). |
| `EntityRepository` | Upsert canonical entities and typed relationship edges. |
| `MismatchRepository` | Upsert and query reasoning findings with evidence and confidence. |
| `SyncRepository` | Read and write connector sync cursors and status. |

## Key types

- `Workspace` — stored workspace record with `id`, `name`, `path`.
- `IngestEvent` — raw source event captured after ingestion with `content_hash` for dedup.
- `ConnectorSync` — replay cursor + status per `(workspace_id, connector, source_uri)`.
