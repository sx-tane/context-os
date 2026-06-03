# chat handler

HTTP handler for local workspace chat queries.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| POST | `/chat/query` | Answers source, status, and findings-intent questions; plugin-backed source questions use live Codex lookup first, then local artifact fallback. |
| POST | `/chat/query/stream` | Streams Codex-style progress logs and heartbeat status over SSE, then emits the final chat answer. |

## Files

- `chat.go` contains `Handler.Query`, `Handler.StreamQuery`, request decoding, SSE progress writing, error mapping, and response mapping from `internal/chat`.

## Behavior

The handler uses `http.MaxBytesReader` for bounded JSON input, runs with a five minute request timeout so Codex-backed live lookups can complete, and delegates intent classification, repository reads, source URI resolution, progress callbacks, and Codex live lookup decisions to `internal/chat.Service`.

`/chat/query/stream` is the preferred UI route. It writes `log` events for lines such as `› Live Codex: GitHub plugin lookup` and `• Starting Codex CLI exec.`, `status` events every two seconds while the lookup is still running, and a final `result` or `error` event. `/chat/query` remains available for non-streaming clients and fallback.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/chat`.
- Preserve workspace scoping for every query.
- Preserve the `provider` response field so the UI can label live Codex answers separately from local DB answers.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
