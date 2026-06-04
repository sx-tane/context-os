# chat handler

HTTP handler for local workspace chat queries.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| POST | `/chat/query` | Answers source, status, and findings-intent questions; plugin-backed source questions use live Codex lookup first, then local artifact fallback. |
| POST | `/chat/query/stream` | Streams Codex-style progress logs and heartbeat status over SSE, then emits the final chat answer. |
| POST | `/chat/session/reset` | Deletes the workspace-scoped stored Codex chat session ID so the next live chat turn starts a fresh Codex conversation. |

## Files

- `chat.go` contains `Handler.Query`, `Handler.StreamQuery`, request decoding, SSE progress writing, error mapping, and response mapping from `internal/runtime/chat`.

## Behavior

The handler uses `http.MaxBytesReader` for bounded JSON input and delegates intent classification, repository reads, source URI resolution, response-language hints, progress callbacks, and Codex live lookup decisions to `internal/runtime/chat.Service`. Live chat queries do not have a fixed API deadline; they run until the client request is canceled or Codex returns, so long plugin lookups do not falsely fail with a handler timeout. Requests may include a single `connector`/`source_uri` or optional `connectors` for multi-source live search; when the service auto-fans out across connected scopes, responses use `connector: "multiple"` and carry concrete source provenance in `answer_sections`.

`/chat/query/stream` is the preferred UI route. It writes `log` events for lines such as `› Live Codex: GitHub plugin lookup` and `• Starting new Codex CLI chat session.` or `• Resuming Codex CLI chat session.`, `status` events every two seconds while the lookup is still running, an early `answer` event when the live answer is ready, and a final `result` or `error` event. Live Codex answers can include `answer_sections`, one structured source card per real source with label, connector, URI, summary, facts, open items, coding notes, links, timestamps, confidence, and status. Concrete live source sections automatically save one Activity artifact per section without running a second connector lookup, including multi-source answers where the aggregate connector is `multiple`. Regex extraction from prose is kept only as a fallback for older unstructured live answers. Connector-only scopes such as `github` or `jira` are skipped when no concrete provenance is visible. `/chat/query` remains available for non-streaming clients and starts eligible evidence saves asynchronously. The frontend refreshes Activity and Graph after `evidence_save_status: "saved"`; Findings stay manual and update only when analysis/findings is run.

Live Codex chat stores only the parsed Codex session ID per workspace under `storage/codex-chat-sessions/`. The first turn starts a new `codex exec --json` conversation, later turns use `codex exec resume --json` with the stored ID, and same-workspace calls are serialized. `POST /chat/session/reset` resolves the workspace and deletes that local pointer only; Codex's global session files are left untouched. `/workspace/reset` and workspace deletion also remove the local pointer.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/runtime/chat`.
- Preserve workspace scoping for every query.
- Preserve the `provider` response field so the UI can label live Codex answers separately from local DB answers.
- Preserve evidence-save metadata so the UI can distinguish `skipped`, `saving`, `saved`, and `error` states.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
