# api

Frontend API client helpers for the Go API and SSE streams.

## Responsibilities

- Keep `/api` request construction, response parsing, request IDs, and network fallback behavior in one place.
- Expose typed helpers for workspace, workspace UI state, artifact, chat, graph, findings, Codex source discovery, login, re-auth, and connector ingest calls.
- Preserve structured error returns for UI workflows so components do not need to parse thrown fetch errors.

## Files

| File | Purpose |
| --- | --- |
| `index.ts` | Public `$lib/api` entrypoint with HTTP helpers and SSE stream readers. |
| `logger.ts` | Quiet-by-default browser request logging, `X-ContextOS-Request-ID` generation, and the `contextosAPITrace(true/false)` browser console helper. |
| `types.ts` | API request/response helper payloads that are not broad app-wide view model types. |

## Maintenance Notes

- Update `apps/frontend/src/lib/README.md` when exported helper contracts change.
- Keep generated OpenAPI types in `../generated/`; do not hand-edit generated declarations.
- `cleanupLiveEvidence` posts to `/artifacts/live-evidence/cleanup` and removes noisy Activity source-event rows only.
- `cleanupGraphNoise` posts to `/graph/cleanup` and permanently removes backend-classified noisy graph entity/relationship rows; source artifacts, chat history, findings, and connected sources remain intact.
- Workspace sync normalization preserves `status`, `event_count`, and `last_error` so source setup can distinguish saved live references, pending sync work, and backend-reported connector errors.
- `getAnalysisBasket`/`putAnalysisBasket` call `/workspace/analysis-basket` for durable selected-evidence state; failed reads return `null` so the route can keep local fallback behavior.
- `getFindingActions`/`putFindingActions` call `/workspace/finding-actions` for durable finding checklist state; failed writes return structured `api_unreachable` errors instead of throwing.
- Browser request logs show the exact frontend API path, method, body preview, response status, duration, and request ID. Match `id=web-...` with API terminal logs when `CONTEXTOS_API_REQUEST_LOGS=1` is enabled.
- Run `bun run test` and `bun run check` after API helper changes.
