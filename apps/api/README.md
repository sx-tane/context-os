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
    slack.go       — POST /slack/ingest, GET /slack/status, GET /slack/connect, GET /slack/callback
  request/
    github.go      — GithubIngest request struct (URI, Token, Provider)
    slack.go       — SlackIngest request struct (URI, Token, Provider)
  response/
    error.go       — WriteJSON, WriteError, WriteConnectorError helpers
    github.go      — GithubIngest response struct
    slack.go       — SlackIngest response struct
  middleware/
    cors.go        — WithCORS middleware
  docs/
    docs.go        — generated: swag init output (do not edit by hand)
    swagger.json   — generated: OpenAPI 2.0 spec
    swagger.yaml   — generated: YAML version of the spec
    api.html       — generated: standalone Redoc HTML (open in browser, no server needed)
```

## Convention: adding a new connector endpoint

1. Add `request/<connector>.go` with the inbound JSON struct.
2. Add `response/<connector>.go` with the outbound JSON struct.
3. Add `handler/<connector>.go` with the HTTP handler — include full swag annotations (see instruction file).
4. Register the route in `main.go` — the `@Router` tag must exactly match.
5. Regenerate docs: `swag init -g apps/api/main.go -o apps/api/docs` (done automatically by `start-all.sh`).

## Endpoints

| Method | Path              | Description                                            |
| ------ | ----------------- | ------------------------------------------------------ |
| GET    | `/health`         | Liveness check — returns `{"status":"ok"}`             |
| POST   | `/github/ingest`  | Ingest a GitHub repo, issue, or PR via MCP             |
| POST   | `/slack/ingest`   | Ingest a Slack channel or message via MCP              |
| GET    | `/slack/status`   | Token availability, source (env/oauth/none), readiness |
| GET    | `/slack/connect`  | Initiates Slack OAuth flow (browser redirect)          |
| GET    | `/slack/callback` | OAuth callback — exchanges code, stores token locally  |
| GET    | `/swagger/`       | Interactive Swagger UI                                 |

## Ingestion providers

Both `/github/ingest` and `/slack/ingest` accept an optional `provider` field:

| Value     | Behaviour                                                                  |
| --------- | -------------------------------------------------------------------------- |
| `"token"` | Default. Uses the direct API connector with the supplied token or env var. |
| `"codex"` | Delegates to the local Codex CLI plugin (requires `codex login`).          |

**Token provider** request:

```json
{ "uri": "https://github.com/owner/repo/issues/1", "token": "ghp_..." }
```

**Codex CLI provider** request (no token needed — Codex handles auth):

```json
{ "uri": "https://github.com/owner/repo/issues/1", "provider": "codex" }
```

When `provider` is `"codex"`, the response `metadata` object includes:

- `codex_log` — full stdout/stderr from the Codex exec run
- `codex_prompt` — the prompt sent to Codex for audit/replay
- `provider` — `"codex_cli"`

**Prerequisites for the Codex provider:**

1. Codex CLI installed: `npm install -g @openai/codex` (done automatically by `start-all.sh`)
2. Plugins enabled: `codex plugin add github@openai-curated` / `slack@openai-curated` (also automatic)
3. Logged in once: `codex login` (local) or `codex login --device-auth` (remote/headless)

## API documentation

The docs are generated automatically from swag annotations in each handler file.

**Interactive UI** (requires API running):

```
http://localhost:8080/swagger/index.html
```

**Standalone HTML** (no server needed — open directly in browser):

```
apps/api/docs/api.html
```

To regenerate after changing a handler or type:

```sh
swag init -g apps/api/main.go -o apps/api/docs
npx @redocly/cli build-docs apps/api/docs/swagger.json --output apps/api/docs/api.html
```

Install `swag` once with:

```sh
go install github.com/swaggo/swag/cmd/swag@latest
```

## Running locally

```sh
go run ./apps/api          # listens on :8080
API_ADDR=:9000 go run ./apps/api
```
