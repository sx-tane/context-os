# graph components

Svelte components for graph-backed context views.

## Components

| File | Purpose |
| --- | --- |
| `GraphPanel.svelte` | Renders graph-oriented workspace context for the frontend. |

## Maintenance Notes

- Keep API fetch and graph data transformation outside low-level rendering components when possible.
- Preserve readable empty, loading, and error states when graph data is unavailable.
- Prefer readable typed relationship maps over dense always-on network diagrams: keep entity navigation compact, keep names directly visible, and draw relationship lines for the selected entity instead of every edge at once.
- Update `internal/graph/README.md` and API handler docs when graph response fields change.
