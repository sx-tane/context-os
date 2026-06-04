# api

Frontend API client helpers for the Go API and SSE streams.

## Responsibilities

- Keep `/api` request construction, response parsing, request IDs, and network fallback behavior in one place.
- Expose typed helpers for workspace, artifact, chat, graph, findings, Codex source discovery, login, re-auth, and connector ingest calls.
- Preserve structured error returns for UI workflows so components do not need to parse thrown fetch errors.

## Files

| File | Purpose |
| --- | --- |
| `index.ts` | Public `$lib/api` entrypoint with HTTP helpers and SSE stream readers. |
| `logger.ts` | Quiet-by-default browser request logging and `X-ContextOS-Request-ID` generation. |

## Maintenance Notes

- Update `apps/frontend/src/lib/README.md` when exported helper contracts change.
- Keep generated OpenAPI types in `../generated/`; do not hand-edit generated declarations.
- Run `bun run test` and `bun run check` after API helper changes.
