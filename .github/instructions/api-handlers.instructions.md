---
description: "Use when creating or modifying HTTP handlers, request types, or response types in the API app. Enforces swag annotations so Swagger docs stay in sync automatically."
applyTo: "apps/api/**/*.go"
---

# API Handler Instructions

## Swagger Annotations — Required on Every Handler

Every exported handler function (`func XxxHandler(w http.ResponseWriter, r *http.Request)`) **must** have a complete swag doc block directly above it. Missing or partial annotations will cause the generated `swagger.json` to be incomplete.

### Mandatory tags

| Tag            | Purpose                                       | Example                                              |
| -------------- | --------------------------------------------- | ---------------------------------------------------- |
| `@Summary`     | One-line description                          | `// @Summary Ingest a GitHub artifact`               |
| `@Description` | Longer explanation                            | `// @Description Fetches a GitHub issue …`           |
| `@Tags`        | Logical group (matches path prefix)           | `// @Tags github`                                    |
| `@Accept`      | Request content type (when body present)      | `// @Accept json`                                    |
| `@Produce`     | Response content type                         | `// @Produce json`                                   |
| `@Param`       | Each path/query/body parameter                | `// @Param body body request.XxxRequest true "desc"` |
| `@Success`     | Happy-path response                           | `// @Success 200 {object} response.XxxResponse`      |
| `@Failure`     | Every error response the handler can return   | `// @Failure 400 {object} map[string]string`         |
| `@Router`      | Path and HTTP method — **must match main.go** | `// @Router /github/ingest [post]`                   |

### Annotation template

```go
// XxxHandler handles METHOD /path/to/route.
//
// @Summary      <one-line summary>
// @Description  <longer explanation>
// @Tags         <tag>
// @Accept       json
// @Produce      json
// @Param        body  body      request.XxxRequest  true  "<description>"
// @Success      200   {object}  response.XxxResponse
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /path/to/route [method]
func XxxHandler(w http.ResponseWriter, r *http.Request) {
```

## Regenerating Docs

Run after every handler or type change:

```bash
swag init -g apps/api/main.go -o apps/api/docs
npx @redocly/cli build-docs apps/api/docs/swagger.json --output apps/api/docs/api.html
```

Both commands run **automatically** via `scripts/start-all.sh` on each local startup.
Install `swag` once with:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

**Outputs after regeneration:**

| File                         | Use                                                         |
| ---------------------------- | ----------------------------------------------------------- |
| `apps/api/docs/swagger.json` | Machine-readable OpenAPI spec                               |
| `apps/api/docs/swagger.yaml` | YAML version of the spec                                    |
| `apps/api/docs/api.html`     | **Standalone HTML — open in any browser, no server needed** |

To view the HTML doc immediately: `open apps/api/docs/api.html` (macOS) or just double-click it.
When the API is running, the interactive UI is also at `http://localhost:8080/swagger/`.

## New Route Checklist

When adding a new route:

1. Create the handler in `apps/api/handler/<name>.go` with full swag annotations.
2. Add the request struct in `apps/api/request/<name>.go`.
3. Add the response struct in `apps/api/response/<name>.go`.
4. Register the route in `apps/api/main.go` — the `@Router` tag must exactly match.
5. Run the two commands above (or just restart via `start-all.sh`).
6. Open `apps/api/docs/api.html` to confirm the new route appears, **or** check `http://localhost:8080/swagger/` if the API is running.

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
- Do not skip `@Failure` tags; every `response.WriteError` call needs a corresponding `@Failure` annotation.
- Do not commit `apps/api/docs/` with stale content; always regenerate before committing.
