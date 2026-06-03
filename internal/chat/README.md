# Internal Chat

Deterministic local chat service for answering workspace-scoped questions from persisted ContextOS repositories.

## Files

| File | Purpose |
| --- | --- |
| `chat.go` | Classifies local chat intent, resolves workspace scope, queries artifacts and sync state, and builds answer summaries. |
| `chat_test.go` | Verifies intent routing, workspace resolution, time range inference, and answer construction. |

## Behavior

The service supports artifact, status, findings, and unsupported intents. It does not call external models or live connectors; answers are built from repository data already stored for the workspace.

```mermaid
flowchart TD
  Q[Chat query] --> W[Resolve workspace]
  W --> S[List sync state]
  S --> I[Classify intent]
  I -->|artifacts| A[Query events]
  I -->|status| T[Summarize syncs]
  I -->|findings| F[Point to findings workflow]
  A --> R[Local answer]
  T --> R
  F --> R
```

## Maintenance Notes

- Keep chat answers deterministic and local-first.
- Preserve workspace scoping before querying artifacts or sync state.
- Update `apps/api/handler/chat/README.md` when service result fields change.
