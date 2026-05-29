# handler/shared

Shared HTTP handler plumbing used by all domain handler packages.

This package is an **internal implementation detail** of `apps/api`. It must not be imported by any code outside `apps/api`.

## Contents

| File             | Responsibility                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `ingest.go`      | Synchronous ingest pipeline: `RunSourceIngest`, `WriteSourceIngest`, `SourceIngestInput`, `NewIngestResponse`, metadata helpers |
| `sse.go`         | SSE infrastructure: `SSEWriter` (with `Write`/`Log`/`Event`/`Error`/`Result`), `SSEHeaders`, `StreamWithHeartbeat`, `StreamCodexIngest[T]` |
| `ingest_test.go` | Unit tests for preview truncation, metadata helpers, capability conversion                                                      |
| `sse_test.go`    | Unit tests for SSE writer concurrency safety and error/result framing                                                           |

## Key exports

- **`SourceIngestInput`** — carries decoded URI, content, cursor, and metadata for a source ingest request.
- **`RunSourceIngest`** — method-guards, decodes JSON body via a caller-supplied decoder, delegates to `WriteSourceIngest`.
- **`WriteSourceIngest`** — validates URI/content, calls `connector.Ingest`, writes JSON response.
- **`NewIngestResponse`** — builds a `response.Ingest` from connector name, capabilities, and ingested events.
- **`SSEWriter`** — concurrency-safe SSE event writer; `Write` emits `event: log` per line while `Event`/`Error`/`Result` serialise status and terminal events through the same mutex so heartbeat and log writes never interleave.
- **`StreamCodexIngest[T]`** — generic SSE handler for any Codex-backed domain request type.
