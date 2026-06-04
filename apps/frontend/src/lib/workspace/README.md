# workspace

Workspace project state and local chat persistence.

## Responsibilities

- Persist workspace selection, connector readiness, project metadata, and chat history in browser storage.
- Keep default and demo workspaces protected from destructive removal.
- Bound cached chat messages, stream lines, source-card sections, and artifact text so local storage stays usable.

## Files

| File | Purpose |
| --- | --- |
| `projectStore.ts` | Svelte stores and helper functions for workspace lifecycle, connector knowledge, chat messages, and backend workspace registration. |

## Maintenance Notes

- Keep cached `answer_sections` bounded and preserve plain `answer` text for backward compatibility.
- Update `apps/frontend/src/lib/README.md` when exported store helpers change.
- Run frontend tests after persistence behavior changes.
