# handler/shared

Shared HTTP handler plumbing used by all domain handler packages.

This package is an **internal implementation detail** of `apps/api`. It must not be imported by any code outside `apps/api`.

## Contents

| File             | Responsibility                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `ingest.go`      | Synchronous ingest pipeline: `RunSourceIngest`, `WriteSourceIngest`, `SourceIngestInput`, `NewIngestResponse`, metadata helpers |
| `persistent_ingest.go` | Persistent ingest service: workspace preparation, pipeline store wiring, sync status updates, audit logging, and persisted ingest responses |
| `sse.go`         | SSE infrastructure: `SSEWriter` (with `Write`/`Log`/`Event`/`Error`/`Result`), `SSEHeaders`, `StreamWithHeartbeat`, `StreamCodexIngest[T]` |
| `ingest_test.go` | Unit tests for preview truncation, metadata helpers, capability conversion                                                      |
| `sse_test.go`    | Unit tests for SSE writer concurrency safety and error/result framing                                                           |

## Key exports

- **`SourceIngestInput`** — carries decoded URI, content, cursor, and metadata for a source ingest request.
- **`RunSourceIngest`** — method-guards, decodes JSON body via a caller-supplied decoder, delegates to `WriteSourceIngest`.
- **`WriteSourceIngest`** — validates URI/content, calls `connector.Ingest`, writes JSON response.
- **`PersistentIngestService`** - runs persistent workspace ingest, records connector sync state, persists pipeline output, and writes audit events.
- **`NewIngestResponse`** — builds a `response.Ingest` from connector name, capabilities, and ingested events.
- **`SSEWriter`** — concurrency-safe SSE event writer; `Write` emits `event: log` per line while `Event`/`Error`/`Result` serialise status and terminal events through the same mutex so heartbeat and log writes never interleave.
- **`StreamCodexIngest[T]`** — generic SSE handler for any Codex-backed domain request type.

## Persistence Notes

Persistent ingest uses a 120 second user-triggered ingest timeout and a detached 30 second write timeout for sync and audit persistence. The detached write context lets final sync state and audit rows finish even when the caller disconnects after the ingest result is produced.
