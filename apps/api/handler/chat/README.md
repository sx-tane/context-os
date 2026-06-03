# chat handler

HTTP handler for local workspace chat queries.

## Endpoint

| Method | Path | Description |
| --- | --- | --- |
| POST | `/chat/query` | Answers local source, status, and findings-intent questions from persisted ContextOS data. |

## Files

- `chat.go` contains `Handler.Query`, request decoding, error mapping, and response mapping from `internal/chat`.

## Behavior

The handler uses `http.MaxBytesReader` for bounded JSON input, runs with a 15 second request timeout, and delegates all intent classification and repository reads to `internal/chat.Service`.

## Maintenance Notes

- Keep the handler thin; local chat behavior belongs in `internal/chat`.
- Preserve workspace scoping for every query.
- Update this README and `apps/api/README.md` when request fields, response fields, or route registration changes.
