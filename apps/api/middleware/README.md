# Middleware

HTTP middleware for the ContextOS API.

## Request Logging

`logging.go` provides `WithRequestLogging`, which wraps registered routes and can emit one start line plus one completion line for each request. Request logs are quiet by default; enable them for local debugging with:

```bash
CONTEXTOS_API_REQUEST_LOGS=1 ./scripts/start-all.sh
```

When enabled, logs include:

- `id` from `X-ContextOS-Request-ID` when the frontend supplied one
- `method`, `path`, registered `route`, and query string
- response `status`, response byte count, and duration

These lines pair with optional frontend console logs from `apps/frontend/src/lib/logger.ts` so a stuck click can be traced from browser request to backend handler receipt.

## WithCORS

`WithCORS` wraps any `http.Handler` with permissive CORS headers so the local SvelteKit frontend can call the Go API directly when the two processes run on different ports.

### Headers set on every response

| Header                         | Value                |
| ------------------------------ | -------------------- |
| `Access-Control-Allow-Origin`  | `*`                  |
| `Access-Control-Allow-Methods` | `GET, POST, OPTIONS` |
| `Access-Control-Allow-Headers` | `Content-Type, X-ContextOS-Request-ID` |

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
