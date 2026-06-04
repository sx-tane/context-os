# insights

Homepage right-pane components for workspace context and insight tabs.

## Components

| Component                 | Purpose                                                                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------- |
| `WorkspaceSummary.svelte` | Renders Codex account, active workspace, ready source summary, and the embedded source setup flow. |
| `FindingsView.svelte`    | Renders Findings rows plus stale/not-run/no-concrete-source empty states from the shared insight status model. |
| `GraphView.svelte`       | Renders the focused selected-entity graph, entity index, relationship details, and type legend.    |
| `ActivityView.svelte`    | Renders source-grouped activity artifacts with a persisted local time filter, hidden scrollbar, click-to-inspect provenance details, and explicit noisy live-evidence cleanup. |

## Data Flow

`+page.svelte` owns API calls, workspace switching, analysis/chat orchestration, selected tab state, and the compact Graph/Findings/Activity status strip. Components receive already-loaded data and call pure helpers from `$lib/findings/viewModel`, `$lib/graph/viewModel`, and `$lib/insights/status` for display shaping.

`WorkspaceSummary.svelte` forwards `KnowledgeInstall.svelte` lifecycle events. Normal source setup uses `done`; destructive reset-all uses `reset` so the route can clear stale findings, graph, activity, selected entity, chat result, and analysis timestamp before refreshing workspace status.

Keep graph behavior focused on the selected entity and its direct links. Do not draw every relationship at once in this surface.

Graph exposes **Clean noisy graph data** as a confirmation-gated action owned by the route. It calls `/graph/cleanup` only after user confirmation, then refreshes workspace status, Activity, and Graph. This permanent cleanup removes low-signal persisted graph rows only; it does not remove source artifacts, chat history, findings, or connected sources.

Findings stay manual. Chat answers and saved live evidence may refresh Activity and Graph immediately, while `FindingsView.svelte` shows `not_run`, `stale`, or `no_concrete_sources` copy from the shared insight status instead of implying that the panel is broken or empty. Connector-only live scopes remain chat-ready but appear as skipped chat-only scopes when Findings needs a concrete repo, project, issue, channel, document, folder, or file.

Activity keeps every fetched artifact inspectable rather than hard-limiting the rendered list. Use the local time-window filter to keep high-volume workspaces readable, and preserve source grouping so users can trace activity back to the exact connector/source record. Expanded events show a readable summary, extracted key lines, links, metadata rows, and a collapsed raw event body so dense live evidence stays inspectable without overwhelming the main row. The **Clean noisy live evidence** action is confirmation-gated through `ConfirmModal` and calls the live-evidence cleanup endpoint; it removes noisy Activity event rows only and must not run automatically during render or refresh.
