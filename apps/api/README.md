# API App

Go API application surface for ContextOS orchestration endpoints.

## Production responsibility

- Expose local-first pipeline orchestration endpoints.
- Return traceable graph and finding results.
- Preserve evidence, confidence, impact, and recommended actions in API responses.
- Keep API contracts aligned with the domain layer.

## Folder layout

```
apps/api/
  main.go          — entry point: addr config, mux, route registration, ListenAndServe only
  handler/
    health.go      — GET /health
    github.go      — POST /github/ingest
  request/
    github.go      — GithubIngest request struct (URI, Token)
  response/
    error.go       — WriteJSON, WriteError, WriteConnectorError helpers
    github.go      — GithubIngest response struct
  middleware/
    cors.go        — WithCORS middleware
```

## Convention: adding a new connector endpoint

1. Add `request/<connector>.go` with the inbound JSON struct.
2. Add `response/<connector>.go` with the outbound JSON struct.
3. Add `handler/<connector>.go` with the HTTP handler function.
4. Register the route in `main.go`.

## Endpoints

| Method | Path             | Description                                |
| ------ | ---------------- | ------------------------------------------ |
| GET    | `/health`        | Liveness check — returns `{"status":"ok"}` |
| POST   | `/github/ingest` | Ingest a GitHub repo, issue, or PR via MCP |

### POST /github/ingest

Request body:

```json
{ "uri": "https://github.com/owner/repo/issues/1", "token": "ghp_..." }
```

- `uri` — required. GitHub URL or `repo://owner/repo/...` URI.
- `token` — optional. GitHub PAT for private repos or higher rate limits. Falls back to `GITHUB_TOKEN` env var.

Response: provenance-rich `document.ingested` event with connector metadata, capabilities, and a content preview.

## Running locally

```sh
go run ./apps/api          # listens on :8080
API_ADDR=:9000 go run ./apps/api
```
