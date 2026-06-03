# chat components

Svelte components for the main local chat thread.

## Components

| File | Purpose |
| --- | --- |
| `ChatPanel.svelte` | Renders the homepage report-agent pane, message evidence previews, findings previews, and composer. |
| `ChatInput.svelte` | Captures and submits the user's chat message. |
| `ChatMessage.svelte` | Renders one chat message and its metadata. |
| `ChatThread.svelte` | Renders the message list and scrolls to the latest message after updates. |

## Usage

These components are presentation-only. API calls, workspace state, and message orchestration should stay in the route or shared frontend helpers such as `$lib/chatController.ts`. Query answer metadata renders `ChatQueryResult.provider` so users can distinguish live Codex lookup from local DB evidence. Pending query copy shows the predicted Live Codex connector/source route, streams Codex-style `›` and `•` progress lines from `/chat/query/stream`, and keeps Local DB visible as the fallback and double-check path.

## Maintenance Notes

- Keep message rendering stable for repeated chat interactions.
- Avoid putting connector ingest logic in these components.
- Update `apps/frontend/src/lib/README.md` if exported chat types or helper contracts change.
