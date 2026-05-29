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
    github.go      — GitHub status and direct/token ingest
    source_connectors.go — Jira direct/Codex ingest and filesystem direct ingest
    slack.go       — Slack status, OAuth, and direct/token ingest
    codex.go       — Codex CLI status, login, and plugin reauth streams
    sse.go         — shared SSE helpers and Codex streaming ingest handlers
  request/
    github.go      — GithubIngest request struct (URI, Token, Provider)
    jira.go        — JiraIngest request struct (URI, token/email/base URL, provider, metadata)
    filesystem.go  — FilesystemIngest request struct (URI/include/exclude/metadata)
    slack.go       — SlackIngest request struct (URI, Token, Provider)
  response/
    error.go       — WriteJSON, WriteError, WriteConnectorError helpers
    github.go      — GithubIngest response struct
    jira.go        — JiraIngest response struct
    filesystem.go  — FilesystemIngest response struct
    slack.go       — SlackIngest response struct
  middleware/
    cors.go        — WithCORS middleware
  docs/
    docs.go        — generated locally by swag init; ignored by git
    swagger.json   — generated locally OpenAPI 2.0 spec; ignored by git
    swagger.yaml   — generated locally YAML spec; ignored by git
    api.html       — generated locally standalone Redoc HTML; ignored by git
```

## Convention: adding a new connector endpoint

1. Add `request/<connector>.go` with the inbound JSON struct.
2. Add `response/<connector>.go` with the outbound JSON struct.
3. Add `handler/<connector>.go` with the HTTP handler — include full swag annotations (see instruction file).
4. Register the route in `main.go` — the `@Router` tag must exactly match.
5. Regenerate docs: `go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g apps/api/main.go -o apps/api/docs` (done automatically by `start-all.sh` when `swag` is installed).

## Endpoints

| Method | Path                    | Description                                            |
| ------ | ----------------------- | ------------------------------------------------------ |
| GET    | `/health`               | Liveness check — returns `{"status":"ok"}`             |
| GET    | `/github/status`        | Checks `GITHUB_TOKEN` and returns account identity     |
| POST   | `/github/ingest`        | Ingest a GitHub repo, issue, PR, or commit via MCP     |
| POST   | `/github/ingest/stream` | Stream Codex-backed GitHub ingest progress over SSE    |
| GET    | `/jira/status`          | Checks Jira environment base URL/token/email readiness |
| POST   | `/jira/ingest`          | Ingest a Jira issue or project via MCP                 |
| POST   | `/jira/ingest/stream`   | Stream Codex/Rovo-backed Jira ingest progress over SSE |
| POST   | `/filesystem/ingest`    | Ingest a local file or folder path via MCP             |
| GET    | `/slack/status`         | Token availability, source (env/oauth/none), readiness |
| GET    | `/slack/connect`        | Initiates Slack OAuth flow (browser redirect)          |
| GET    | `/slack/callback`       | OAuth callback — exchanges code, stores token locally  |
| POST   | `/slack/ingest`         | Ingest a Slack channel or message via MCP              |
| POST   | `/slack/ingest/stream`  | Stream Codex-backed Slack ingest progress over SSE     |
| GET    | `/codex/status`         | Codex CLI install/login/plugin status                  |
| POST   | `/codex/login`          | Run `codex login --device-auth` and stream logs as SSE |
| POST   | `/codex/plugin-reauth`  | Re-add `github`, `atlassian-rovo`, or `slack` plugin   |
| GET    | `/swagger/`             | Interactive Swagger UI                                 |

Full request/response schemas are in the interactive docs — see below.

GitHub, Jira, and Slack ingest requests accept `provider`. Use `"token"` or omit it for direct API-token ingestion. Use `"codex"` for Codex CLI plugin ingestion; streaming clients should call the matching `/ingest/stream` endpoint.

Jira and filesystem direct request fields:

- Jira accepts `uri`, `token`, `email`, `api_base_url`, `expand`, `content`, `cursor`, `provider`, and `metadata`. The token/email/base URL fields map to connector metadata and fall back to `JIRA_TOKEN`, `JIRA_EMAIL`, and `JIRA_BASE_URL`. `provider=codex` routes through `atlassian-rovo@openai-curated`.
- Filesystem normally needs only `uri`, which may be a file or folder path. A directory `uri` is walked recursively and returns one event per supported child file. Optional advanced fields include inline `content`, `cursor`, `include`, `exclude`, and `metadata`; include/exclude map to explicit path rules before local file reads. Folder guardrails can be set with metadata keys `filesystem_max_files` and `filesystem_max_file_size`; defaults are `1000` files and `10485760` bytes per file. Filesystem also handles spreadsheet extraction and OpenAPI JSON/YAML summary metadata.

Filesystem responses keep the existing first-event fields (`event`, `preview`, and `metadata`) and also include aggregate fields (`events`, `previews`, `metadata_items`, and `event_count`) for folder ingestion.

## API documentation

The docs are generated automatically from swag annotations in each handler file.
The generated files under `apps/api/docs/` are ignored by git, so regenerate them after a clean checkout before running the API or `go test ./...`.

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
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g apps/api/main.go -o apps/api/docs
npx @redocly/cli build-docs apps/api/docs/swagger.json --output apps/api/docs/api.html
```

Install `swag` once with:

```sh
go install github.com/swaggo/swag/cmd/swag@latest
```

After installing `swag`, the shorter command is:

```sh
swag init -g apps/api/main.go -o apps/api/docs
```

## Running locally

```sh
go run ./apps/api          # listens on :8080
API_ADDR=:9000 go run ./apps/api
```
