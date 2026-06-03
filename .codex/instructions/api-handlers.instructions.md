---
description: "Use when creating or modifying HTTP handlers, request types, or response types in the API app."
applyTo: "apps/api/**/*.go"
---

# API Handler Instructions

## Skill

For a full step-by-step guide, skeletons, and a completion checklist, apply the **contextos-api-handler** skill.

## Package Layout

Each connector is its own sub-package: `apps/api/handler/<name>/<name>.go`.
Do **not** put handler files directly under `apps/api/handler/`.

## Handler Shape

Every exported handler must:

1. Check `r.Method` as the **first statement** and return 405 for unsupported methods.
2. Decode the request body with `http.MaxBytesReader(w, r.Body, limit)` — never `r.Body` directly.
3. Delegate ingest logic to `shared.RunSourceIngest` or `shared.WriteSourceIngest` — no direct connector calls.
4. Use `response.WriteError(w, status, "snake_code", "message")` for all errors.
5. Use `response.WriteJSON(w, http.StatusOK, payload)` for all success responses.
6. Carry full swag annotations (`@Summary`, `@Tags`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`).

## New Route Checklist

1. Create `apps/api/handler/<name>/<name>.go` with `Status`, `Ingest`, and `IngestStream` handlers.
2. Create `apps/api/handler/<name>/<name>_test.go` — apply the **go-test-patterns** skill.
3. Add `<Name>Ingest` request struct to `apps/api/request/ingest.go`.
4. Reuse `response.Ingest` from `apps/api/response/ingest.go`.
5. Register all three routes in `apps/api/main.go` with `cors: true`.
6. Update `apps/api/README.md` and nearest handler/source README when routes, env vars, metadata, or setup changes.
7. Run `go build ./...` and `go test ./apps/api/handler/<name>/...`.

## Do Not

- Do not put business logic inside handler files — delegate to `internal/` packages.
- Do not bypass `shared.RunSourceIngest` to call connectors directly.
- Do not leave endpoint tables or setup docs stale when adding routes.
