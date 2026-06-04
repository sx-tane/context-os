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
- **`PersistentIngestService`** - runs persistent workspace ingest, records connector sync state, persists pipeline output, writes audit events, and can store live-answer evidence while deriving graph state without findings.
- **`WithPersistentRelationshipAssistant`** — optional persistent-ingest wiring for validated relationship assistance; nil keeps deterministic relationships.
- **`WithPersistentGraphVerifier`** — optional live-evidence follow-up that verifies cross-source graph relationships from Local DB snapshots.
- **`NewIngestResponse`** — builds a `response.Ingest` from connector name, capabilities, and ingested events.
- **`SSEWriter`** — concurrency-safe SSE event writer; `Write` emits `event: log` per line while `Event`/`Error`/`Result` serialise status and terminal events through the same mutex so heartbeat and log writes never interleave.
- **`StreamCodexIngest[T]`** — generic SSE handler for any Codex-backed domain request type.

## Persistence Notes

Persistent ingest uses a 120 second user-triggered ingest timeout and a detached 30 second write timeout for sync and audit persistence. The detached write context lets final sync state and audit rows finish even when the caller disconnects after the ingest result is produced.

`PersistEvents` is the full streaming persistence path: it stores emitted events and then runs normalization, extraction, identity, relationship, graph, and reasoning stages from `internal/stages/*`. `PersistEvidenceEvents` is narrower for live chat answers: it writes the returned answer as a local Activity artifact plus sync/audit state, then runs normalization, extraction, identity, relationship, and graph persistence on those already-returned events. It intentionally skips reasoning so Findings do not auto-run from live chat saves. When `WithPersistentGraphVerifier` is configured, live-evidence saves also run a Local DB-only cross-source verifier after graph persistence.

Relationship assistance is not enabled here by default. Bootstrap may pass
`WithPersistentRelationshipAssistant` when `CONTEXTOS_AI_RELATIONSHIPS=codex`; otherwise the
relationship stage uses deterministic rules only.

Graph verification is not enabled here by default. Bootstrap may pass `WithPersistentGraphVerifier`
when `CONTEXTOS_GRAPH_VERIFIER` is set; otherwise live evidence saves use only normal graph-only
pipeline output.
