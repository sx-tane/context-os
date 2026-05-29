# handler/shared

Shared HTTP handler plumbing used by all domain handler packages.

This package is an **internal implementation detail** of `apps/api`. It must not be imported by any code outside `apps/api`.

## Contents

| File             | Responsibility                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `ingest.go`      | Synchronous ingest pipeline: `RunSourceIngest`, `WriteSourceIngest`, `SourceIngestInput`, `NewIngestResponse`, metadata helpers |
| `sse.go`         | SSE infrastructure: `SSEWriter`, `SSEHeaders`, `SSEError`, `SSEResult`, `StreamWithHeartbeat`, `StreamCodexIngest[T]`           |
| `ingest_test.go` | Unit tests for preview truncation, metadata helpers, capability conversion                                                      |

## Key exports

- **`SourceIngestInput`** — carries decoded URI, content, cursor, and metadata for a source ingest request.
- **`RunSourceIngest`** — method-guards, decodes JSON body via a caller-supplied decoder, delegates to `WriteSourceIngest`.
- **`WriteSourceIngest`** — validates URI/content, calls `connector.Ingest`, writes JSON response.
- **`NewIngestResponse`** — builds a `response.Ingest` from connector name, capabilities, and ingested events.
- **`SSEWriter`** — `io.Writer` that emits `event: log` SSE events per line.
- **`StreamCodexIngest[T]`** — generic SSE handler for any Codex-backed domain request type.
