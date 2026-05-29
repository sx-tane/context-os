# Middleware

HTTP middleware for the ContextOS API.

## WithCORS

`WithCORS` wraps any `http.Handler` with permissive CORS headers so the local SvelteKit frontend can call the Go API directly when the two processes run on different ports.

### Headers set on every response

| Header                         | Value                |
| ------------------------------ | -------------------- |
| `Access-Control-Allow-Origin`  | `*`                  |
| `Access-Control-Allow-Methods` | `GET, POST, OPTIONS` |
| `Access-Control-Allow-Headers` | `Content-Type`       |

`OPTIONS` preflight requests are short-circuited with `204 No Content` — the inner handler is not called.

### Usage

Applied in `apps/api/main.go` per-route via the `cors` field on the route entry:

```go
// route registration (main.go)
{pattern: "/slack/connect", handler: http.HandlerFunc(handler.SlackConnect), cors: true},

// registerRoutes wires it conditionally
if r.cors {
    handler = middleware.WithCORS(handler)
}
mux.Handle(r.pattern, handler)
```

Only routes with `cors: true` receive the header. Most API routes are called through the SvelteKit `/api` proxy and do not need it.
