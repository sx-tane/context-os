# chat components

Svelte components for the main local chat thread.

## Components

| File | Purpose |
| --- | --- |
| `ChatInput.svelte` | Captures and submits the user's chat message. |
| `ChatMessage.svelte` | Renders one chat message and its metadata. |
| `ChatThread.svelte` | Renders the message list and scrolls to the latest message after updates. |

## Usage

These components are presentation-only. API calls, workspace state, and message orchestration should stay in the route or shared frontend helpers.

## Maintenance Notes

- Keep message rendering stable for repeated chat interactions.
- Avoid putting connector ingest logic in these components.
- Update `apps/frontend/src/lib/README.md` if exported chat types or helper contracts change.
