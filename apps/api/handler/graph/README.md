# graph handler

HTTP handler for querying persisted workspace entity graph data.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| GET | `/graph` | Returns canonical entities for a workspace, optionally filtered by entity type. |

## Query Parameters

| Parameter | Required | Notes |
| --- | --- | --- |
| `workspace_id` | Yes | May be a workspace path or stored workspace ID. |
| `entity_type` | No | Filters canonical entities by type, such as `feature`, `person`, or `service`. |

## Files

- `graph.go` contains `Handler.Query`, workspace path resolution, and entity repository lookup.

## Maintenance Notes

- Keep graph reads backed by `repository.EntityRepository`.
- Preserve deterministic response fields: `workspace_id`, `entity_type`, `entity_count`, and `entities`.
- Update `internal/graph/README.md` if graph persistence or entity contracts change.
