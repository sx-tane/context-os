# chat handler

HTTP handler for local workspace chat queries.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| POST | `/chat/query` | Answers source, status, and findings-intent questions; plugin-backed source questions use live Codex lookup first, then local artifact fallback. |
| POST | `/chat/query/stream` | Streams Codex-style progress logs and heartbeat status over SSE, then emits the final chat answer. |

## Files

- `chat.go` contains `Handler.Query`, `Handler.StreamQuery`, request decoding, SSE progress writing, error mapping, and response mapping from `internal/runtime/chat`.

## Behavior

The handler uses `http.MaxBytesReader` for bounded JSON input and delegates intent classification, repository reads, source URI resolution, progress callbacks, and Codex live lookup decisions to `internal/runtime/chat.Service`. Live chat queries do not have a fixed API deadline; they run until the client request is canceled or Codex returns, so long plugin lookups do not falsely fail with a handler timeout.

`/chat/query/stream` is the preferred UI route. It writes `log` events for lines such as `› Live Codex: GitHub plugin lookup` and `• Starting Codex CLI exec.`, `status` events every two seconds while the lookup is still running, an early `answer` event when the live answer is ready, and a final `result` or `error` event. Live Codex answers can include `answer_sections`, one structured source card per real source with label, connector, URI, summary, facts, open items, coding notes, links, timestamps, confidence, and status. Concrete live source sections automatically save one Activity artifact per section without running a second connector lookup. Regex extraction from prose is kept only as a fallback for older unstructured live answers. Connector-only scopes such as `github` or `jira` are skipped when no concrete provenance is visible. `/chat/query` remains available for non-streaming clients and starts eligible evidence saves asynchronously. The frontend refreshes Activity and Graph after `evidence_save_status: "saved"`; Findings stay manual and update only when analysis/findings is run.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/runtime/chat`.
- Preserve workspace scoping for every query.
- Preserve the `provider` response field so the UI can label live Codex answers separately from local DB answers.
- Preserve evidence-save metadata so the UI can distinguish `skipped`, `saving`, `saved`, and `error` states.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
