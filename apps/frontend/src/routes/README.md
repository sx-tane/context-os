# Frontend Routes

This folder contains Svelte route files and page entrypoints.

## Files

- `+page.svelte`: main local ContextOS UI page that probes service status, renders the workspace switcher, opens source setup, coordinates local chat and analysis, and composes the Findings / Graph / Activity insight tabs from `$lib/components/insights`.

## Current UI Shape

The main route keeps a two-pane workspace layout: chat on the left and project insights on the right. The top bar exposes the active workspace, new workspace creation, source setup, and service status.

Controls should use the same restrained mono theme:

- Workspace select and new workspace input use underline-style fields, not boxed cards.
- Workspace `Remove` opens a confirmation dialog before destructive cleanup. The default and demo workspaces cannot be removed. Confirming removal calls `DELETE /workspace?path=...` first; only a successful backend delete clears local project/chat/source state, closes setup state, clears graph/activity state, and moves to the next saved workspace. If the API delete fails, the workspace remains visible and the route reports the backend failure in chat.
- The demo workspace uses local seed data for sources, findings, graph, activity, and chat/source queries so users can inspect the intended experience without ingesting live sources or requiring a backend demo workspace row.
- Source, Clear, Send, Run Analysis, and workspace action buttons use the same underline button treatment and change color on hover.
- The topbar uses the same 16px horizontal inset as the main content panes so workspace controls, source setup, and status read as part of one layout.
- The footer console strip shows a live local clock, service status, graph links, and finding count; keep its timer route-owned and cleaned up on destroy.
- The chat and insight panes can be resized from the center divider on desktop; the layout returns to one column below the compact breakpoint.
- The embedded source setup panel expands within the right pane without a nested visible scrollbar.
- Findings / Graph / Activity uses the segmented tab style already defined in the route.
- The Codex account is shown once under `CODEX`; do not repeat the connected status line or label it as a generic profile unless it is actually a user profile surface.
- The insight summary shows the active workspace name once under `WORKSPACE`; do not repeat the workspace path below it.
- Activity rows show whether each artifact came from `LOCAL` filesystem ingest or a plugin-backed `SOURCE`, then show connector, source URI, and ingest time.
- Chat source questions show a neutral source-context loading state; plugin-backed concrete source links and saved sources use Codex-backed live lookup first, while filesystem questions remain local DB first. When a streamed live answer reports saved evidence, the route refreshes workspace state so Activity shows the new artifact immediately; Graph and Findings still update only after Run Analysis.
- Graph view renders a readable typed relationship map, not a full always-on hairball network: the left index shows selected, linked, and top entities as compact flat rows with optional filtering, while the selected entity is drawn as a focused graph with incoming/outgoing relationship lines.
- The graph entity type summary is API-driven and lives in the right detail panel with counts and stable generated colors.
- Relationship context is selective: the focused graph draws links for the selected entity instead of all graph links at once. The detail panel groups incoming and outgoing relationships by relationship kind.
- Run Analysis preserves backend JSON errors and surfaces frontend API connectivity failures as an explicit message to start `scripts/start-all.sh` or check the `/api` proxy.
- Run Analysis aggregates successful findings from every ready source, shows per-source failures inline, and uses an explicit zero-finding message when analysis completed without mismatch signals.
- Findings render as flat rows with separated severity/title, detected/evidence times, description, and recommended action blocks. Findings are displayed in English by default, and all source text in this route is kept in English.

## Responsibilities

- Define page-level data flow and actions.
- Compose workspace switching, source setup, chat, graph, findings, and activity views.
- Keep insight tab rendering in `$lib/components/insights` and pure display helpers in `$lib/*ViewModel.ts` so the route remains focused on orchestration.
- Delegate chat command/query execution to `$lib/chatController.ts` and analysis source execution to `$lib/analysisRunner.ts`; the route should only wire Svelte state/store callbacks.
- Keep route state transitions understandable and testable.

## Maintenance Checklist

- Document significant route behavior changes here.
- Keep route integration tests aligned with UI behavior.
- Update linked component docs for new props or events.
