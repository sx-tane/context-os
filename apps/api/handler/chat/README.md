# chat handler

HTTP handler for local workspace chat queries.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| POST | `/chat/query` | Answers source, status, and findings-intent questions from persisted ContextOS data, with optional Codex-backed live lookup for configured source-specific questions. |

## Files

- `chat.go` contains `Handler.Query`, request decoding, error mapping, and response mapping from `internal/chat`.

## Behavior

The handler uses `http.MaxBytesReader` for bounded JSON input, runs with a 150 second request timeout so Codex-backed live lookups can complete, and delegates intent classification, repository reads, and optional Codex live lookup decisions to `internal/chat.Service`.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/chat`.
- Preserve workspace scoping for every query.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
