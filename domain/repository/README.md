# domain/repository

Domain repository interfaces for workspace-scoped persistence.

Implementations live in `internal/store`. Nothing in `domain/` depends on `internal/`.

## Interfaces

| Interface | Purpose |
|-----------|---------|
| `WorkspaceRepository` | Register and retrieve workspace records by path. |
| `EventRepository` | Upsert and query raw ingested source events (idempotent by `id+workspace_id`). Duplicate upserts update the stored row and return the written row count. Query supports connector, source URI, date range, text, and limit filters. |
| `EntityRepository` | Upsert and list canonical entities and typed relationship edges. |
| `MismatchRepository` | Upsert and query reasoning findings with evidence and confidence. |
| `SyncRepository` | Read and write connector sync cursors and status. |

## Key types

- `Workspace` — stored workspace record with `id`, `name`, `path`.
- `IngestEvent` — raw source event captured after ingestion with `content_hash` for dedup.
- `EventQuery` — workspace-scoped artifact filter used by `/artifacts` and local chat queries.
- `ConnectorSync` — replay cursor + status per `(workspace_id, connector, source_uri)`. `status="connected"` represents a saved external source reference with no local ingest cursor or event count yet.

## Graph Reads

`EntityRepository.ListRelationships` returns persisted relationship edges for a
workspace and can optionally scope the result to a set of entity IDs. The graph
API uses this to render source-backed entity links without re-running analysis.
