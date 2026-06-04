# workspace

Workspace project state and local chat persistence.

## Responsibilities

- Persist workspace selection, connector readiness, project metadata, and chat history in browser storage.
- Keep default and demo workspaces protected from destructive removal.
- Bound cached chat messages, stream lines, source-card sections, and artifact text so local storage stays usable.
- Preserve saved connector sources when backend sync rows are `connected` or `pending`; only backend `error` rows downgrade a saved source to an error state.

## Files

| File | Purpose |
| --- | --- |
| `projectStore.ts` | Svelte stores and helper functions for workspace lifecycle, connector knowledge, chat messages, and backend workspace registration. |
| `statusMapping.ts` | Pure sync-row to connector-knowledge reconciliation used by `projectStore.ts`. |

## Maintenance Notes

- Keep cached `answer_sections` bounded and preserve plain `answer` text for backward compatibility.
- `loadWorkspaceStatus` reconciles local connector knowledge with backend sync rows: `connected` and `pending` live references stay ready without fake event counts, concrete rows with events keep counts, and `error` rows surface `last_error` on the connector.
- Update `apps/frontend/src/lib/README.md` when exported store helpers change.
- Run frontend tests after persistence behavior changes.
