---
description: "Use when creating or modifying HTTP handlers, request types, or response types in the API app."
applyTo: "apps/api/**/*.go"
---

# API Handler Instructions

## Handler Comments

Every exported handler function (`func XxxHandler(w http.ResponseWriter, r *http.Request)`) should have a short comment that names the route and behavior. Keep comments useful for readers first.

## New Route Checklist

When adding a new route:

1. Create the handler in `apps/api/handler/<name>.go` with a clear route/behavior comment.
2. Add the request struct to `apps/api/request/ingest.go` unless the connector needs a separate request domain.
3. Reuse `response.Ingest` from `apps/api/response/ingest.go` unless the connector needs a distinct response contract.
4. Register the route in `apps/api/main.go`.
5. Run `go test ./...`.

## Handler Method Guard

Every handler must check `r.Method` at the top and return `405` for unsupported methods:

```go
if r.Method != http.MethodPost {
    response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
    return
}
```

## Do Not

- Do not put business logic inside handler files — delegate to `internal/` packages.
- Do not leave endpoint tables or setup docs stale when adding routes.
