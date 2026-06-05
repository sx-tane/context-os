# chat

Chat orchestration helpers and demo workspace fixtures.

## Responsibilities

- Route chat commands into source setup, findings analysis, clear history/session memory, or source queries.
- Keep streaming chat state out of Svelte components by managing `ChatMessage.stream` updates in helpers.
- Preserve local demo workspace behavior for product inspection without live connectors, including planning-first agent notes and structured source-card answers.

## Files

| File | Purpose |
| --- | --- |
| `controller.ts` | Builds chat messages, infers live source routes, calls stream/fallback chat APIs, and refreshes workspace data after saved evidence. |
| `demoWorkspace.ts` | Provides deterministic demo workspace artifacts, findings, graph, status, and chat answers. |

## Maintenance Notes

- Keep source-card data in `ChatQueryResult.answer_sections`; do not parse answer prose in the frontend.
- Keep `response_language` tied to the user's actual question language. English questions with short CJK source terms such as `決済GW` should stay English, while Chinese/Japanese/Korean questions should keep their requested language.
- Keep `clear` local-first: browser chat is cleared even when the best-effort `/chat/session/reset` call cannot reach the API.
- Send `connectors` for broad prompts and explicit source-connector wording such as `all source connectors`, `allowed source connectors`, `connected source connectors`, or `my source connectors`, using ready non-filesystem workspace sources as the allow-list for backend live fanout even when the prompt also mentions a concrete connector name.
- Send `mode` with every query. `auto` is the default Codex-then-Local route, `codex` keeps answers live-only, and `local` keeps the query inside persisted Local DB evidence.
- Keep demo chat scenarios table-driven or helper-driven as they grow. Demo prompts should cover planning mode, agent mode, functions/notes, sources/source cards, findings, graph, Activity cleanup/filtering, evidence basket, analysis preview, checklist, export, and stream behavior without calling the backend.
- Keep long stream transcripts bounded before they enter persisted project state.
- Run `bun run test` for chat controller behavior changes.
