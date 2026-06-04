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
| `include_noise` | No | Set to `true` to include low-signal regex entities and low-confidence `co_occurs_in_document` links for debugging old persisted rows. |

## Files

- `graph.go` contains `Handler.Query`, workspace path resolution, and entity repository lookup.
- `graph_test.go` verifies `/graph` returns the flat entity shape consumed by the frontend graph tab.

## Response Shape

`/graph` returns flat `GraphEntity` rows, not raw `CanonicalEntity` domain objects. By default it hides low-signal regex-only rows and low-confidence co-occurrence links; `include_noise=true` returns the legacy/noisy view. The response includes visible counts and hidden totals for frontend compatibility:

```json
{
  "workspace_id": "workspace-id",
  "count": 1,
  "entity_count": 1,
  "relationship_count": 0,
  "filtered_entity_count": 2,
  "filtered_relationship_count": 3,
  "total_entity_count": 3,
  "total_relationship_count": 3,
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
- Preserve deterministic response fields: `workspace_id`, `entity_type`, `entity_count`, `relationship_count`, filtered counts, and `entities`.
- Update `internal/graph/README.md` if graph persistence or entity contracts change.
