# workspace handler

HTTP handlers for registering, resetting, deleting, listing, inspecting, and saving local ContextOS workspace UI state.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/workspace` | Lists registered workspaces. |
| POST | `/workspace/upsert` | Creates or updates a workspace by local path. |
| POST | `/workspace/source` | Saves a connector/source URI as a connected source reference without ingesting content. |
| DELETE | `/workspace?path=...` | Deletes DB-backed workspace memory, parsed JSON, graph snapshots, Codex chat session metadata, and the workspace row without recreating it. |
| POST | `/workspace/reset` | Deletes DB-backed workspace memory, parsed JSON, graph snapshots, and Codex chat session metadata, then recreates an empty workspace row. |
| GET | `/workspace/status` | Returns event, entity, relationship, mismatch, audit, and connector sync counts for one workspace. |
| GET/PUT | `/workspace/analysis-basket` | Reads or replaces the workspace-scoped analysis evidence basket. |
| GET/PUT | `/workspace/finding-actions` | Reads or replaces the workspace-scoped finding action checklist. |

## Files

- `workspace.go` contains `Handler`, repository wiring, request decoding, workspace reset/delete, local artifact cleanup, status aggregation, and typed UI-state validation.
- `workspace_test.go` verifies reset behavior, status response behavior, and workspace-scoped UI-state round trips.

## Maintenance Notes

- Keep workspace operations bounded by `workspaceRequestTimeout`.
- Use `repository.WorkspaceResetter` only for reset-capable stores.
- Connected external source setup writes `connector_syncs` with `status="connected"`, `event_count=0`, and `last_synced_at=nil`; it does not ingest content or create findings.
- UI-state endpoints require a workspace path or ID, store JSON by stable state keys, and return empty typed payloads when no row has been saved yet.
- `analysis_basket` items require `id`, `connector`, `uri`, `label`, `origin`, and `addedAt`; optional fields preserve the source artifact or chat message that added the item.
- `finding_actions` items require `findingId`, `status`, and `updatedAt`; supported statuses are `open`, `checking`, `done`, `ignored`, and `false_positive`.
- `DELETE /workspace` verifies the workspace row is gone before returning success; the frontend should not clear local state when this endpoint fails.
- Reset/delete cleanup removes `storage/parsed/<workspace_id>/`, `storage/snapshots/<workspace_id>.json`, `storage/snapshots/<workspace_id>_*.json`, and the stored Codex chat session pointer under `storage/codex-chat-sessions/` when local artifact/session directories are configured.
- Keep detailed status counts best-effort so missing optional repositories do not hide the core workspace status.
- Update `apps/api/README.md` when endpoint paths, request fields, or response fields change.
