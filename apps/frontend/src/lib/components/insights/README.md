# insights

Homepage right-pane components for workspace context and insight tabs.

## Components

| Component                 | Purpose                                                                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------- |
| `WorkspaceSummary.svelte` | Renders Codex account, active workspace, ready source summary, and the embedded source setup flow. |
| `FindingsView.svelte`    | Renders the Findings tab rows, zero-finding state, and source/entity counts from analysis output.  |
| `GraphView.svelte`       | Renders the focused selected-entity graph, entity index, relationship details, and type legend.    |
| `ActivityView.svelte`    | Renders source-grouped activity artifacts with a persisted local time filter, hidden scrollbar, and click-to-inspect provenance details. |

## Data Flow

`+page.svelte` owns API calls, workspace switching, analysis/chat orchestration, and selected tab state. These components receive already-loaded data and call pure helpers from `$lib/findings/viewModel` and `$lib/graph/viewModel` for display shaping.

`WorkspaceSummary.svelte` forwards `KnowledgeInstall.svelte` lifecycle events. Normal source setup uses `done`; destructive reset-all uses `reset` so the route can clear stale findings, graph, activity, selected entity, chat result, and analysis timestamp before refreshing workspace status.

Keep graph behavior focused on the selected entity and its direct links. Do not draw every relationship at once in this surface.

Activity keeps every fetched artifact inspectable rather than hard-limiting the rendered list. Use the local time-window filter to keep high-volume workspaces readable, and preserve source grouping so users can trace activity back to the exact connector/source record.
