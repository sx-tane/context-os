# workspace handler

HTTP handlers for registering, resetting, listing, and inspecting local ContextOS workspaces.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/workspace` | Lists registered workspaces. |
| POST | `/workspace` | Creates or updates a workspace by local path. |
| POST | `/workspace/reset` | Deletes DB-backed workspace memory and recreates an empty workspace row. |
| GET | `/workspace/status` | Returns event, entity, relationship, mismatch, audit, and connector sync counts for one workspace. |

## Files

- `workspace.go` contains `Handler`, repository wiring, request decoding, workspace reset, and status aggregation.
- `workspace_test.go` verifies reset behavior and status response behavior.

## Maintenance Notes

- Keep workspace operations bounded by `workspaceRequestTimeout`.
- Use `repository.WorkspaceResetter` only for reset-capable stores.
- Keep detailed status counts best-effort so missing optional repositories do not hide the core workspace status.
- Update `apps/api/README.md` when endpoint paths, request fields, or response fields change.
