# Internal Sync

Background sync marker for connector sync records.

## Files

| File | Purpose |
| --- | --- |
| `worker.go` | Runs a periodic pass over workspaces and marks stale local sync rows or errored connector syncs as pending. |

## Behavior

The worker lists workspaces, loads connector sync records, and marks records pending when they are already in `error` status or have local sync state that is stale. A `connected` row with no `LastSyncedAt`, cursor, or events is a saved live source reference; it stays `connected` until a user-triggered ingest or explicit error changes it. Full re-ingest still happens through user-triggered ingest paths.

```mermaid
flowchart TD
  T[Ticker] --> W[List workspaces]
  W --> S[List sync records]
  S --> C{Error?}
  C -->|yes| P[Mark pending]
  C -->|no| L{Has local sync state?}
  L -->|no| N[Leave connected reference]
  L -->|yes| H{Stale?}
  H -->|yes| P
  H -->|no| N[Leave unchanged]
```

## Maintenance Notes

- Keep the worker cancellation-aware through the caller-provided context.
- Do not trigger live connector ingest from this package without updating repository contracts and API status docs.
- Treat `connected` as a saved external source reference and `pending` as a needs-sync marker, not as a disconnected source.
- Update workspace status docs when sync status values change.
