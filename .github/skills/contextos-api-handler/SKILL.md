---
name: contextos-api-handler
description: "Create a new ContextOS API handler and its paired source connector following the established pattern. Use when: adding a new /connector/status or /connector/ingest route; adding a new internal/source/<name> connector; pairing a new source with an HTTP handler. Covers package layout, Status/Ingest/IngestStream shape, shared.RunSourceIngest usage, swagger annotations, request/response types, main.go route registration, and test coverage."
argument-hint: "What is the connector name (e.g. notion, googledrive)?"
user-invocable: true
---

# ContextOS API Handler Skill

## Outcome

Deliver a fully-wired, tested handler for a new source connector with:

- `apps/api/handler/connectors/<name>/` — Status, Ingest, IngestStream handlers
- `apps/api/request/` — request struct for the new connector
- `apps/api/main.go` — route registration for all three routes
- `internal/source/<name>/` — MCP source connector (if not yet present)
- Relevant `README.md` files updated for routes, setup, connector behavior, and run commands

---

## Decision Points

| Situation                                                    | Action                                                                               |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| Source connector already exists in `internal/source/<name>/` | Skip connector creation; write handler only                                          |
| Connector has OAuth flow (like Slack)                        | Add Connect + Callback handlers, add `/connect` and `/callback` routes               |
| Connector has Codex plugin                                   | Pass `codexsource.MetadataPlugin` = `codexsource.Plugin<Name>` to connector metadata |
| No Codex plugin                                              | Use direct MCP connector; set `provider` field in request type                       |

---

## Procedure

1. **Create the request type** in `apps/api/request/ingest.go`.
   Add a `<Name>Ingest` struct with JSON tags and `example:` annotations. See pattern in [handler skeleton](./assets/handler-skeleton.md).

2. **Create the handler package** at `apps/api/handler/connectors/<name>/<name>.go`.
   - Package doc comment: `// Package <name> provides HTTP handlers for the /<name>/* routes.`
   - `Status` — GET only; read env vars; return `response.WriteJSON` map.
   - `Ingest` — POST only; decode via `json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))`; delegate to `shared.RunSourceIngest` or `shared.WriteSourceIngest`.
   - `IngestStream` — POST only; call `shared.SSEHeaders`; call `shared.RunSourceIngestStream`.
   - Add full swag annotations to every handler.

3. **Create handler tests** at `apps/api/handler/connectors/<name>/<name>_test.go`.
   Apply the **go-test-patterns** skill. Minimum tests:
   - `TestStatusMethodNotAllowed` — non-GET → 405
   - `TestStatusReturnsDisconnectedWhenNoEnvVar` — no env → connected=false
   - `TestIngestMethodNotAllowed` — non-POST → 405
   - `TestIngestRejectsMalformedJSON` — bad JSON → 400

4. **Create (or update) the source connector** at `internal/source/<name>/<name>.go`.
   - Wrap `source.NewMCPConnector("<name>", contracts.Capability<X>)`.
   - Implement `Name()`, `Capabilities()`, and `Ingest()`.
   - `Ingest` must: clone metadata, validate URI, call `c.base.Ingest`, return events.
   - Apply the **go-best-practices** skill.

5. **Register routes** in `apps/api/main.go`:

   ```go
   {pattern: "/<name>/status",        handler: http.HandlerFunc(<name>.Status),       cors: true},
   {pattern: "/<name>/ingest",        handler: http.HandlerFunc(<name>.Ingest),        cors: true},
   {pattern: "/<name>/ingest/stream", handler: http.HandlerFunc(<name>.IngestStream),  cors: true},
   ```

   Add the import `<name> "context-os/apps/api/handler/connectors/<name>"`.

6. **Update documentation**:
   - Update `apps/api/README.md` when routes, setup, env vars, or Swagger/codegen flow changes.
   - Update the nearest handler or source README when connector behavior, metadata, or replay expectations change.
   - Keep endpoint tables and setup docs aligned with the new route names.

7. **Run checks**:
   ```bash
   go build ./...
   go test ./apps/api/handler/connectors/<name>/... ./internal/source/<name>/...
   go vet ./apps/api/... ./internal/source/<name>/...
   swag init -g apps/api/main.go -o apps/api/docs --parseDependency
   ```

---

## Handler Shape Rules

- **Method guard is always the first statement** in every handler.
- Use `response.WriteError(w, status, "code_snake_case", "human message")` for all error responses.
- Use `response.WriteJSON(w, http.StatusOK, payload)` for all success responses.
- Body size limit for ingest: `8 << 20` (8 MB). For status: no body.
- Use `shared.RunSourceIngest` when no Codex branching is needed. Use `shared.WriteSourceIngest` directly when you need to build custom metadata before the call.
- Never call the source connector directly from a handler — always go through `shared`.

---

## Source Connector Shape Rules

- Package doc: `// Package <name> provides an MCP source connector for <Name> artifacts.`
- Constructor: `func NewConnector() contracts.MCPSourceConnector`
- `Ingest` must clone metadata with `cloneMetadata` before mutating.
- Connector errors use `contracts.ConnectorError` via `source.NewConnectorError` or `c.base.Ingest` error wrapping.
- Idempotency: same `URI` + `Content` → same `event.ID` (guaranteed by `events.New`).

---

## References

- [Handler Skeleton](./assets/handler-skeleton.md) — copy-paste Go files for handler + source connector
- [Checklist](./references/handler-checklist.md) — review before marking done
- Real examples:
  - Handler: [`apps/api/handler/connectors/github/github.go`](../../../../apps/api/handler/connectors/github/github.go)
  - Handler: [`apps/api/handler/connectors/jira/jira.go`](../../../../apps/api/handler/connectors/jira/jira.go)
  - Shared ingest: [`apps/api/handler/shared/ingest.go`](../../../../apps/api/handler/shared/ingest.go)
  - Source connector: [`internal/source/jira/jira.go`](../../../../internal/source/jira/jira.go)
  - Route registration: [`apps/api/main.go`](../../../../apps/api/main.go)
