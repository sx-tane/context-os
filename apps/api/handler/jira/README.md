# handler/jira

HTTP handlers for the `/jira/*` routes.

## Handlers

| Function       | Route                 | Method | Description                                                                |
| -------------- | --------------------- | ------ | -------------------------------------------------------------------------- |
| `Status`       | `/jira/status`        | GET    | Reports whether `JIRA_BASE_URL`, `JIRA_TOKEN`, `JIRA_EMAIL` are configured |
| `Ingest`       | `/jira/ingest`        | POST   | Ingests a Jira issue or project via native or Codex/Rovo connector         |
| `IngestStream` | `/jira/ingest/stream` | POST   | Streams Codex Atlassian Rovo plugin progress as SSE, then emits result     |

## Request type

`request.JiraIngest` — fields: `URI`, `Content`, `Cursor`, `Provider`, `Token`, `Email`, `APIBaseURL`, `Expand`, `Metadata`.

## Private helpers

- `buildMetadata(req)` — clones `req.Metadata` and injects `jira_token`, `jira_email`, `jira_api_base_url`, `jira_expand`.
