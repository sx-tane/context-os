# workspace handler

HTTP handlers for registering, resetting, deleting, listing, and inspecting local ContextOS workspaces.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/workspace` | Lists registered workspaces. |
| POST | `/workspace/upsert` | Creates or updates a workspace by local path. |
| POST | `/workspace/source` | Saves a connector/source URI as a connected source reference without ingesting content. |
| DELETE | `/workspace?path=...` | Deletes DB-backed workspace memory and the workspace row without recreating it. |
| POST | `/workspace/reset` | Deletes DB-backed workspace memory and recreates an empty workspace row. |
| GET | `/workspace/status` | Returns event, entity, relationship, mismatch, audit, and connector sync counts for one workspace. |

## Files

- `workspace.go` contains `Handler`, repository wiring, request decoding, workspace reset/delete, and status aggregation.
- `workspace_test.go` verifies reset behavior and status response behavior.

## Maintenance Notes

- Keep workspace operations bounded by `workspaceRequestTimeout`.
- Use `repository.WorkspaceResetter` only for reset-capable stores.
- Connected external source setup writes `connector_syncs` with `status="connected"`, `event_count=0`, and `last_synced_at=nil`; it does not ingest content or create findings.
- `DELETE /workspace` verifies the workspace row is gone before returning success; the frontend should not clear local state when this endpoint fails.
- Keep detailed status counts best-effort so missing optional repositories do not hide the core workspace status.
- Update `apps/api/README.md` when endpoint paths, request fields, or response fields change.
