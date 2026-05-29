# handler/github

HTTP handlers for the `/github/*` routes.

## Handlers

| Function       | Route                   | Method | Description                                             |
| -------------- | ----------------------- | ------ | ------------------------------------------------------- |
| `Status`       | `/github/status`        | GET    | Reports connected GitHub account (reads `GITHUB_TOKEN`) |
| `Ingest`       | `/github/ingest`        | POST   | Ingests a GitHub artifact via native or Codex connector |
| `IngestStream` | `/github/ingest/stream` | POST   | Streams Codex CLI progress as SSE, then emits result    |

## Request type

`request.GithubIngest` — fields: `URI`, `Provider` (`""` or `"codex"`), `Token` (optional override).

## Private helpers

- `resolveUser(token)` — calls `api.github.com/user` to return `(login, name)`.
