# domain/repository

Domain repository interfaces for workspace-scoped persistence.

Implementations live in `internal/persistence/store`. Nothing in `domain/` depends on `internal/`.

## Interfaces

| Interface | Purpose |
|-----------|---------|
| `WorkspaceRepository` | Register and retrieve workspace records by path. |
| `WorkspaceUIStateRepository` | Read and replace durable workspace-scoped frontend workflow state documents. |
| `EventRepository` | Upsert and query raw ingested source events (idempotent by `id+workspace_id`). Duplicate upserts update the stored row and return the written row count. Query supports connector, source URI, date range, text, and limit filters. |
| `EventDeleter` | Optional delete capability for explicit workspace-scoped cleanup flows such as noisy live-chat evidence removal. |
| `EntityRepository` | Upsert and list canonical entities and typed relationship edges. |
| `GraphEvidenceDeleter` | Optional delete capability for graph rows tied to selected source event IDs. |
| `GraphNoiseCleaner` | Optional explicit delete capability for backend-classified low-signal graph rows. |
| `MismatchRepository` | Upsert and query reasoning findings with evidence and confidence. |
| `SyncRepository` | Read and write connector sync cursors and status. |

## Key types

- `Workspace` — stored workspace record with `id`, `name`, `path`.
- `IngestEvent` — raw source event captured after ingestion with `content_hash` for dedup.
- `EventQuery` — workspace-scoped artifact filter used by `/artifacts` and local chat queries.
- `ConnectorSync` — replay cursor + status per `(workspace_id, connector, source_uri)`. `status="connected"` represents a saved external source reference with no local ingest cursor or event count yet.
- `WorkspaceUIState` — raw JSON payload stored by `(workspace_id, state_key)` for typed handler-owned UI documents such as `analysis_basket` and `finding_actions`.

## Graph Reads

`EntityRepository.ListRelationships` returns persisted relationship edges for a
workspace and can optionally scope the result to a set of entity IDs. The graph
API uses this to render source-backed entity links without re-running analysis.

`GraphEvidenceDeleter.DeleteGraphEvidenceByEventIDs` is for explicit Activity
delete flows. It prunes graph rows whose provenance points at the same selected
event IDs, keeping Activity cleanup scoped instead of running broad graph noise
cleanup.

`GraphNoiseCleaner.CleanupGraphNoise` is separate from graph reads. It is only
for user-confirmed cleanup flows and permanently deletes low-signal graph rows;
it must not delete source artifacts, reasoning findings, chat history, connector
syncs, or workspace records.

## Workspace UI State

`WorkspaceUIStateRepository` deliberately stores raw JSON because the API handler
owns validation for each state document. Repository implementations should not
interpret analysis basket or finding action fields; they only enforce workspace
scope, state key scope, replacement semantics, and updated timestamps.
