# chat components

Svelte components for the main local chat thread.

## Components

| File | Purpose |
| --- | --- |
| `ChatPanel.svelte` | Renders the homepage report-agent pane, compact stream progress, source traces, message evidence previews, findings previews, and composer. |
| `ChatInput.svelte` | Captures and submits the user's chat message. |
| `ChatMessage.svelte` | Renders one chat message and its metadata. |
| `ChatThread.svelte` | Renders the message list and scrolls to the latest message after updates. |

## Usage

These components are presentation-only. API calls, workspace state, and message orchestration should stay in the route or shared frontend helpers such as `$lib/chatController.ts`. Query answer metadata renders `ChatQueryResult.provider` so users can distinguish live Codex lookup from local DB evidence.

Pending query progress lives in `ChatMessage.stream` instead of the answer text. `ChatPanel.svelte` renders it as a collapsed stream block by default: running and completed streams show status, the latest Codex-style `›` or `•` progress line, and the final summary when available. Expanding the block shows the full stream transcript in order.

The composer uses a wrapping textarea so long prompts move onto new lines instead of stretching horizontally. Enter inserts a line break, the send button submits, and Ctrl/Cmd+Enter submits from the keyboard.

Query answers always render the answer first, then a lightweight Local DB save status when the backend reports `evidence_save_status`, then a source trace with provider, connector, source URI, non-duplicative stream summary, and artifact count. Artifact evidence details remain available when `chatResult.artifacts.length > 0`. Live Codex answers without artifacts are labeled as a live source trace; a separate `Local DB: saved ...`, `skipped ...`, or `save failed ...` line tells the user whether the live answer evidence was persisted.

The message list follows new messages only while the user is already near the bottom of the thread. If the user scrolls upward during a long Codex stream, the pane preserves their viewport instead of jumping to the newest progress line.

## Maintenance Notes

- Keep message rendering stable for repeated chat interactions.
- Avoid putting connector ingest logic in these components.
- Update `apps/frontend/src/lib/README.md` if exported chat types or helper contracts change.
