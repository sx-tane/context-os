# API Handler Checklist

Use before merging any new handler or source connector change.

---

## Request Type

- [ ] `<Name>Ingest` struct added to `apps/api/request/ingest.go`.
- [ ] All fields have `json:"..."` tags.
- [ ] All fields have `example:"..."` tags for swagger.

---

## Handler Package (`apps/api/handler/<name>/`)

- [ ] Package doc comment: `// Package <name> provides HTTP handlers for the /<name>/* routes.`
- [ ] `Status` is GET-only; method guard is the **first statement**.
- [ ] `Ingest` is POST-only; method guard is the **first statement**.
- [ ] `IngestStream` is POST-only; method guard is the **first statement**.
- [ ] Body decoded with `http.MaxBytesReader(w, r.Body, limit)` — never `r.Body` directly.
- [ ] Ingest delegates to `shared.RunSourceIngest` or `shared.WriteSourceIngest` — no direct connector calls.
- [ ] All error responses use `response.WriteError(w, status, "snake_code", "message")`.
- [ ] All success responses use `response.WriteJSON(w, http.StatusOK, payload)`.
- [ ] Full swag annotations on every exported handler function.

---

## Handler Tests (`apps/api/handler/<name>/<name>_test.go`)

- [ ] Package is `package <name>_test` (external).
- [ ] Every `func Test*` has a doc comment.
- [ ] `TestStatusMethodNotAllowed` — POST to Status → 405.
- [ ] `TestStatusReturnsDisconnectedWhenNoEnvVar` — no env → connected:false.
- [ ] `TestIngestMethodNotAllowed` — GET to Ingest → 405.
- [ ] `TestIngestRejectsMalformedJSON` — bad body → 400.
- [ ] `TestIngestStreamMethodNotAllowed` — GET to IngestStream → 405.
- [ ] Assertions use `recorder.Code` with `t.Fatalf("Handler() status = %d, want %d", ...)`.

---

## Source Connector (`internal/source/<name>/`)

- [ ] Package doc: `// Package <name> provides an MCP source connector for <Name> artifacts.`
- [ ] `NewConnector()` returns `contracts.MCPSourceConnector`.
- [ ] `Ingest` clones metadata before mutating.
- [ ] No direct imports between `internal/` stage packages.
- [ ] Connector errors use `contracts.ConnectorError` (not bare `fmt.Errorf`).

---

## Route Registration (`apps/api/main.go`)

- [ ] `/status`, `/ingest`, and `/ingest/stream` routes all registered.
- [ ] All three routes have `cors: true`.
- [ ] Import alias matches other handlers (e.g., `<name> "context-os/apps/api/handler/<name>"`).

---

## Documentation

- [ ] `apps/api/README.md` updated when routes, env vars, setup, or Swagger/codegen flow changes.
- [ ] Nearest handler or source connector README updated when connector behavior, metadata, or replay expectations change.
- [ ] Endpoint tables and setup docs do not mention stale paths or removed handlers.

---

## Build & Test

- [ ] `go build ./...` passes with no errors.
- [ ] `go test ./apps/api/handler/<name>/... ./internal/source/<name>/...` passes.
- [ ] `go vet ./...` produces no warnings.
- [ ] `swag init` succeeds and regenerates `docs/` cleanly.
