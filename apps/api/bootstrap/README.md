# API Bootstrap

Composes API route registration and DB-backed handler wiring for the local ContextOS service.

## Files

- `bootstrap.go`: builds the `http.ServeMux`, registers public routes, and creates DB-backed workspace, artifact, graph, presentation, chat, and sync services when a SQL connection is available.
- `bootstrap_test.go`: verifies route registration and DB-optional handler behavior.

## Behavior

Static connector, Codex, health, and Swagger routes are always registered. Connector and Codex HTTP handlers are imported from `apps/api/handler/connectors/<name>` while their source implementations remain in `internal/source/<name>`. Workspace, artifact, graph, and chat routes are registered when `Routes` receives a database handle because those routes need persisted workspace state. The chat route uses the live Codex answerer and the shared persistent ingest service so concrete live source answers can save evidence into the Local DB.

Relationship assistance is disabled by default. Set `CONTEXTOS_AI_RELATIONSHIPS=codex` to wire a cached local Codex CLI relationship assistant into persistent ingest. Accepted proposals are still validated by `internal/relationship`, and failed Codex calls fall back to deterministic relationship edges.

```mermaid
flowchart TD
  DB[(SQL DB)] --> B[bootstrap.Routes]
  B --> S[Persistent stores]
  ENV[CONTEXTOS_AI_RELATIONSHIPS=codex] --> A[Cached Codex relationship assistant]
  A --> S
  S --> H[DB-backed handlers]
  H --> M[HTTP mux]
  C[Static connector handlers] --> M
```

## Maintenance Notes

- Keep route additions in `Routes` covered by bootstrap tests.
- Keep DB-backed service construction here rather than inside individual handlers.
- Update `apps/api/README.md` when public route behavior changes.
- Keep relationship assistance behind an explicit env flag; do not make Codex calls part of default ingest.
