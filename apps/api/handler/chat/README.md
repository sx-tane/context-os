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

`/chat/query/stream` is the preferred UI route. It writes `log` events for lines such as `› Live Codex: GitHub plugin lookup` and `• Starting Codex CLI exec.`, `status` events every two seconds while the lookup is still running, an early `answer` event when the live answer is ready, and a final `result` or `error` event. Concrete live source answers such as Jira issue keys, Jira browse URLs, GitHub repositories, Slack channels, and docs automatically save the returned live answer as local evidence through the existing persistence pipeline without running a second connector lookup; connector-only scopes such as `github` or `jira` are skipped to avoid broad implicit ingestion. `/chat/query` remains available for non-streaming clients and starts eligible evidence saves asynchronously. The frontend refreshes Activity after `evidence_save_status: "saved"`; Graph and Findings are derived outputs and require a separate analysis run.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/chat`.
- Preserve workspace scoping for every query.
- Preserve the `provider` response field so the UI can label live Codex answers separately from local DB answers.
- Preserve evidence-save metadata so the UI can distinguish `skipped`, `saving`, `saved`, and `error` states.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
