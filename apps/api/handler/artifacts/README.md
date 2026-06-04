# artifacts handler

HTTP handler for querying persisted source artifacts in a workspace.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| GET | `/artifacts` | Returns workspace-scoped artifacts from persisted ingest events. |
| POST | `/artifacts/live-evidence/cleanup` | Explicitly removes old noisy `live_chat_answer` artifacts created from duplicate full answers, URL path fragments, or generic terms. |

## Query Parameters

| Parameter | Required | Notes |
| --- | --- | --- |
| `workspace_id` or `workspace_path` | Yes | Resolves by workspace path, ID, or stored workspace row. |
| `connector` | No | Filters by normalized connector name. |
| `source_uri` | No | Filters by channel, repository, folder, document, or source URI. |
| `q` | No | Text search over stored artifact content. |
| `since` or `after` | No | RFC3339 inclusive lower bound. |
| `until` or `before` | No | RFC3339 exclusive upper bound. |
| `limit` | No | Defaults to 20 and caps at 100. |

## Files

- `artifacts.go` contains `Handler.Query`, `Handler.CleanupLiveEvidence`, workspace resolution, event query construction, query parameter parsing, and the conservative noisy-live-evidence selector.

## Maintenance Notes

- Keep artifact responses backed by `repository.EventRepository`; do not re-query live connectors from this handler.
- Keep live evidence cleanup explicit; it must not auto-run during artifact queries.
- Preserve the 10 second request timeout unless repository query behavior changes.
- Update `apps/api/README.md` when route registration, request parameters, or response shape changes.
