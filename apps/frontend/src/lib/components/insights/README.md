# insights

Homepage right-pane components for workspace context and insight tabs.

## Components

| Component                 | Purpose                                                                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------- |
| `WorkspaceSummary.svelte` | Renders Codex account, active workspace, ready source summary, and the embedded source setup flow. |
| `FindingsView.svelte`    | Renders the Findings tab rows, zero-finding state, and source/entity counts from analysis output.  |
| `GraphView.svelte`       | Renders the focused selected-entity graph, entity index, relationship details, and type legend.    |
| `ActivityView.svelte`    | Renders recent source artifacts with local/source origin, connector/provider labels, and timestamps. |

## Data Flow

`+page.svelte` owns API calls, workspace switching, analysis/chat orchestration, and selected tab state. These components receive already-loaded data and call pure helpers from `$lib/findingsViewModel.ts` and `$lib/graphViewModel.ts` for display shaping.

Keep graph behavior focused on the selected entity and its direct links. Do not draw every relationship at once in this surface.
