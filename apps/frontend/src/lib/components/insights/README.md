# insights

Homepage right-pane components for workspace context and insight tabs.

## Components

| Component                 | Purpose                                                                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------- |
| `WorkspaceSummary.svelte` | Renders Codex account, active workspace, ready source summary, and the embedded source setup flow. |
| `AnalysisPreview.svelte` | Renders the exact source list Run Analysis will use, basket selections, available unselected evidence, skipped chat-only scopes, source health, and Markdown export. |
| `FindingsView.svelte`    | Renders Findings rows plus stale/not-run/no-concrete-source empty states from the shared insight status model. |
| `GraphView.svelte`       | Renders the focused selected-entity graph, entity index, relationship details, and type legend.    |
| `ActivityView.svelte`    | Renders source-grouped activity artifacts with time, connector, source URI, evidence-type, and keyword filters; click-to-inspect provenance details; Ask/Pin evidence actions; explicit selected/visible Activity deletion with matching graph-evidence pruning; and noisy live-evidence cleanup. |

## Data Flow

`+page.svelte` owns API calls, workspace switching, cancelable analysis/chat orchestration, selected tab state, and the compact Graph/Findings/Activity status strip. Components receive already-loaded data and call pure helpers from `$lib/findings/viewModel`, `$lib/graph/viewModel`, `$lib/workflow/viewModel`, and `$lib/insights/status` for display shaping.

`WorkspaceSummary.svelte` forwards `KnowledgeInstall.svelte` lifecycle events. Normal source setup uses `done`; destructive reset-all uses `reset` so the route can clear stale findings, graph, activity, selected entity, chat result, and analysis timestamp before refreshing workspace status.

Keep graph behavior focused on the selected entity and its direct links. Do not draw every relationship at once in this surface.

Graph exposes **Clean noisy graph data** as a confirmation-gated action owned by the route. It calls `/graph/cleanup` only after user confirmation, then refreshes workspace status, Activity, and Graph. This permanent cleanup removes backend-classified low-signal persisted graph rows only; it does not remove source artifacts, chat history, findings, or connected sources. The selected node details panel can also request manual entity deletion through the route-owned confirmation flow; that calls `/graph/entity` for one entity, removes relationships touching it, refreshes the graph, and leaves source artifacts intact.

`AnalysisPreview.svelte` receives a prebuilt `$lib/workflow/viewModel` model from the route. It must match the actual `runAnalysis` input: when the evidence basket has items, included rows are basket-only; otherwise included rows come from concrete Sources plus chat/Activity-derived evidence. Available-but-not-selected rows stay visible so users understand what the basket is narrowing. Source Health uses transparent rows and connector-colored source titles, not status cards.

Findings stay manual. Chat answers and saved live evidence may refresh Activity and Graph immediately, while `FindingsView.svelte` shows `not_run`, `stale`, or `no_concrete_sources` copy from the shared insight status instead of implying that the panel is broken or empty. Connector-only live scopes remain chat-ready but appear as skipped chat-only scopes when Findings needs a concrete repo, project, issue, channel, document, folder, or file. The route owns Run Analysis, Cancel Analysis, persistent finding action saves, and copy/share behavior; insight components receive already-loaded state and do not start or abort analysis themselves.

Finding checklist controls are intentionally small: each finding row can cycle `open -> checking -> done`, copy a share-ready text block, hide an item as `ignored`, mark an incorrect item as `false_positive`, reopen hidden items from the status filter, and show source evidence values or URLs. Persisted state is owned by the route and backend workspace UI-state endpoints. The default Findings filter shows active findings only, so ignored and false-positive rows stay out of the main review flow without deleting source evidence.

Activity keeps every fetched artifact inspectable rather than hard-limiting the rendered list. Use one sticky toolbar with filter controls first and compact actions second; do not repeat the Activity title/count inside the toolbar because the route tab/status strip already names the view and count. Preserve source grouping so users can trace activity back to the exact connector/source record, but prefer human-readable source labels: Drive rows should show file names when available, and Slack rows should show channel or conversation names before falling back to raw URLs. Compact rows and expanded events stay transparent, use restrained connector-colored titles to match chat source sections, and render the summary, extracted key lines, and collapsed raw event body with the shared safe Markdown block so bullets, headings, links, inline code, and bold text stay readable. Expanded events may prefill chat with `connector:source_uri` context, pin the concrete evidence to the analysis basket, or delete the selected local Activity event, but they must not auto-send chat or trigger analysis. The toolbar must not expose bulk visible-row delete. The cleanup action is confirmation-gated through `ConfirmModal` and calls the live-evidence cleanup endpoint; it removes backend-classified noisy Activity event rows only and must not run automatically during render or refresh.
