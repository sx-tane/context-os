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
- `graph_test.go` verifies `/graph` returns the flat entity shape consumed by the frontend graph tab.

## Response Shape

`/graph` returns flat `GraphEntity` rows, not raw `CanonicalEntity` domain objects. The response includes both `count` and `entity_count` for frontend compatibility:

```json
{
  "workspace_id": "workspace-id",
  "count": 1,
  "entity_count": 1,
  "entities": [
    {
      "id": "entity-id",
      "name": "Refund status",
      "type": "requirement",
      "source": "github://repo/pull/1",
      "confidence": 0.91,
      "evidence": ["github://repo/pull/1"]
    }
  ]
}
```

## Maintenance Notes

- Keep graph reads backed by `repository.EntityRepository`.
- Preserve deterministic response fields: `workspace_id`, `entity_type`, `entity_count`, and `entities`.
- Update `internal/graph/README.md` if graph persistence or entity contracts change.
