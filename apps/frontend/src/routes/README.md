# Frontend Routes

This folder contains Svelte route files and page entrypoints.

## Files

- `+page.svelte`: main local ContextOS UI page that probes service status, renders the workspace switcher, opens source setup, coordinates local chat and analysis, and composes the Findings / Graph / Activity insight tabs from `$lib/components/insights`.

## Current UI Shape

The main route keeps a two-pane workspace layout: chat on the left and project insights on the right. The top bar exposes the active workspace, new workspace creation, source setup, and service status.

Controls should use the same restrained mono theme:

- Workspace select and new workspace input use underline-style fields, not boxed cards.
- Workspace `Remove` opens the shared `ui/ConfirmModal` before destructive cleanup. The default and demo workspaces cannot be removed. Confirming removal calls `DELETE /workspace?path=...` first; only a successful backend delete clears local project/chat/source state, closes setup state, clears graph/activity state, and moves to the next saved workspace. If the API delete fails, the workspace remains visible and the route reports the backend failure in chat.
- Source setup `Reset all data` calls `/workspace/reset` for every known workspace, clears local project/chat storage, then emits a reset lifecycle event. The route must immediately clear `lastFindings`, analysis timestamp, graph data, selected entity, activity artifacts, latest chat result, and workspace status before refreshing from the empty backend state.
- The demo workspace uses local seed data for sources, findings, graph, activity, workflow state, and chat/source queries so users can inspect the intended experience without ingesting live sources or requiring a backend demo workspace row. Opening the demo seeds a planning-first assistant note when demo chat is empty; the note uses structured source cards to expose planning mode, agent chat mode, source cards, Activity cleanup/filtering, stream behavior, findings, graph, evidence basket, analysis preview, finding checklist, Markdown export, and source setup prompts before live connector setup.
- The walkthrough includes a **Workflow Demo** shortcut. It opens the protected demo workspace, expands Analysis Preview, loads demo basket/checklist state, and keeps the Findings tab visible so basket-only analysis input, source health, finding action statuses, copy/share, and export are immediately visible.
- Source, Clear, Send, Run Analysis, Cancel Analysis, and workspace action buttons use the same underline button treatment and change color on hover.
- Preview sits beside Run Analysis and renders the same concrete source model used by `runAnalysis`. If the evidence basket has pinned items, Run Analysis uses basket-only input while Preview still shows other available concrete sources as unselected.
- Evidence Basket and Finding Action Checklist state are durable backend workspace UI state loaded through `$lib/api`, not browser-only project store state. Switching, creating, deleting, or resetting workspaces must clear stale local workflow state and reload the active workspace state.
- Export Markdown is generated in the route from already-loaded source health, analysis preview/basket, findings/checklist, graph counts, and recent Activity. It does not call the worker.
- The topbar uses the same 16px horizontal inset as the main content panes so workspace controls, source setup, and status read as part of one layout.
- The footer status line distinguishes API health from DB-backed workspace route availability. `/health` can be online while Local DB is unavailable; in that state source setup and workspace-backed actions must show Local DB unavailable instead of treating Codex as the blocker.
- The footer console strip reuses the shared insight status labels for Activity evidence, Graph availability, and manual Findings freshness. Keep idle route state static so the main page does not re-render every second while no analysis or ingest is running.
- Workspace refreshes reuse the status response already loaded into the project store, ignore stale overlapping refreshes, and skip graph/artifact requests until the workspace has at least one ready source.
- The chat and insight panes default to a compact chat rail with a wider insight workspace and can be resized from the center divider on desktop; the layout returns to one column below the compact breakpoint.
- The embedded source setup panel expands within the right pane without a nested visible scrollbar.
- Findings / Graph / Activity uses the segmented tab style already defined in the route.
- Switching between Findings, Graph, and Activity closes the analysis preview so stale preview context does not stay open above a different tab.
- The Codex account is shown once under `CODEX`; do not repeat the connected status line or label it as a generic profile unless it is actually a user profile surface.
- The insight summary shows the active workspace name once under `WORKSPACE`; do not repeat the workspace path below it.
- Activity rows show whether each artifact came from `LOCAL` filesystem ingest or a plugin-backed `SOURCE`, then show connector, source URI, and ingest time. Activity keeps the route-provided time window plus connector, source URI, evidence-type, and keyword filters in the sticky toolbar, with clear, delete-visible, and cleanup actions aligned in the same control row when space allows.
- Chat source questions show a neutral source-context loading state plus a composer mode switch. `Auto` uses Codex-backed live lookup first and Local DB fallback, `Codex` keeps the query live-only, and `Local` skips live lookup. Plugin-backed concrete source links and saved concrete sources use Codex first in Auto mode, while filesystem questions remain local DB first. Connector-only source rows such as `github:github` can run broad connected-account live chat, but they only become saved Activity evidence when the answer returns concrete provenance. Long live lookups keep API/Codex status in a checking state instead of flipping to offline from transient probe timeouts. When a streamed live answer reports saved evidence, the route refreshes workspace state so Activity and Graph can show newer evidence immediately; Findings stay manual and display a stale/not-run/no-concrete-source status until Run Analysis covers that evidence.
- Chat source cards and expanded Activity events can prefill chat with source context or pin the evidence for analysis. Prefill never auto-sends, and pinning does not start analysis by itself.
- Activity receives confirmation-gated cleanup and selected-delete callbacks from the route. The route calls `POST /artifacts/live-evidence/cleanup` for backend-classified noise or `POST /artifacts/delete` for user-selected/visible rows, refreshes workspace Activity afterward, and never triggers cleanup automatically during normal refresh. These actions remove local Activity rows only and do not delete upstream source data.
- Graph view renders a readable typed relationship map, not a full always-on hairball network: the left index shows selected, linked, and top entities as compact flat rows with optional filtering, while the selected entity is drawn as a focused graph with incoming/outgoing relationship lines.
- The graph entity type summary is API-driven and lives in the right detail panel with counts and stable generated colors.
- Relationship context is selective: the focused graph draws links for the selected entity instead of all graph links at once. The detail panel groups incoming and outgoing relationships by relationship kind.
- Graph receives a separate confirmation-gated cleanup action from the route. The route calls `POST /graph/cleanup`, then refreshes workspace status, Activity, and Graph. This permanent cleanup removes backend-classified low-signal persisted graph rows only; source artifacts, chat history, findings, and connected sources are not deleted.
- Run Analysis preserves backend JSON errors and surfaces frontend API connectivity failures as an explicit message to start `scripts/start-local.sh` or check the `/api` proxy. While analysis is running, the route shows a same-style Cancel Analysis button that aborts the current source request and stops queued sources.
- Run Analysis aggregates successful findings from every concrete ready source, plus concrete sources derived from the latest chat result and Activity artifacts. If the basket has items, Run Analysis narrows to those basket concrete sources only. Broad live connector rows remain chat-only scopes and are reported separately.
- The insight status strip derives one shared Activity / Graph / Findings model from `$lib/insights/status`, including concrete analysis-ready sources from Sources, chat evidence, Activity evidence, and basket selections versus chat-only live connector scopes.
- Findings render as flat rows with separated severity/title, detected/evidence times, description, recommended action blocks, compact action-checklist status, copy/share control, and evidence/source links. Findings are displayed in English by default, and all source text in this route is kept in English.

## Responsibilities

- Define page-level data flow and actions.
- Compose workspace switching, source setup, chat, graph, findings, and activity views.
- Keep insight tab rendering in `$lib/components/insights` and pure display helpers in `$lib/findings/viewModel` plus `$lib/graph/viewModel` so the route remains focused on orchestration.
- Delegate chat command/query execution to `$lib/chat/controller` and analysis source execution to `$lib/findings/analysisRunner`; the route should only wire Svelte state/store callbacks.
- Keep route state transitions understandable and testable.

## Maintenance Checklist

- Document significant route behavior changes here.
- Keep route integration tests aligned with UI behavior.
- Update linked component docs for new props or events.
