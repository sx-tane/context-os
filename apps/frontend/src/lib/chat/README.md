# chat

Chat orchestration helpers and demo workspace fixtures.

## Responsibilities

- Route chat commands into source setup, findings analysis, clear history, or source queries.
- Keep streaming chat state out of Svelte components by managing `ChatMessage.stream` updates in helpers.
- Preserve local demo workspace behavior for product inspection without live connectors.

## Files

| File | Purpose |
| --- | --- |
| `controller.ts` | Builds chat messages, infers live source routes, calls stream/fallback chat APIs, and refreshes workspace data after saved evidence. |
| `demoWorkspace.ts` | Provides deterministic demo workspace artifacts, findings, graph, status, and chat answers. |

## Maintenance Notes

- Keep source-card data in `ChatQueryResult.answer_sections`; do not parse answer prose in the frontend.
- Keep long stream transcripts bounded before they enter persisted project state.
- Run `bun run test` for chat controller behavior changes.
