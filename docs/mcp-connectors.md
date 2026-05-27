# MCP Connector Architecture

All external source integrations in ContextOS are MCP-first connectors. Each connector implements the domain `MCPSourceConnector` contract and converts source-specific input into `document.ingested` events.

## Phase 1 — local-first connectors

These connectors ingest from local paths or authenticated APIs with no hosted infrastructure dependency.

| Connector  | Source                                                                                                 | Capability    | Issue |
| ---------- | ------------------------------------------------------------------------------------------------------ | ------------- | ----- |
| GitHub     | Repository, issues, PRs                                                                                | `repository`  | #7    |
| Slack      | Messages, threads, channels                                                                            | `messages`    | #8    |
| Jira       | Issues, comments, status history                                                                       | `issues`      | #9    |
| OpenAPI    | Endpoint and schema specs                                                                              | `api_spec`    | #10   |
| Excel      | Workbooks, sheets, cells                                                                               | `spreadsheet` | #11   |
| Filesystem | Local files (`.txt`, `.md`, `.go`, `.yaml`, `.json`, `.ts`, `.docx`, `.pdf`, `.pptx`, `.xlsx`, `.csv`) | `files`       | #12   |

## Phase 2 — cloud and knowledge-base connectors

These connectors require OAuth or API token credentials and target cloud-hosted knowledge stores.

| Connector             | Source                                           | Capability | Issue |
| --------------------- | ------------------------------------------------ | ---------- | ----- |
| Google Drive          | Google Docs, Sheets, Slides                      | `files`    | #30   |
| SharePoint / OneDrive | Word, Excel, PowerPoint, PDF via Microsoft Graph | `files`    | #31   |
| Confluence            | Pages and spaces (Cloud and Data Center)         | `docs`     | #32   |
| Notion                | Pages and database entries                       | `docs`     | #33   |

## Connector output

Each connector emits raw ingestion events that are then normalized, classified, extracted, resolved, related, stored in the context graph, and analyzed for delivery mismatches.

## HTTP API surface

Connectors are exposed via the Go API (`apps/api`). Each connector has a dedicated endpoint under `/<connector>/ingest`.

| Method | Path             | Connector | Description                        |
| ------ | ---------------- | --------- | ---------------------------------- |
| POST   | `/github/ingest` | GitHub    | Ingest a repo, issue, or PR by URI |
| POST   | `/slack/ingest`  | Slack     | Ingest a channel or message by URI |

Request body:

```json
{ "uri": "https://github.com/owner/repo/issues/1", "token": "ghp_..." }
```

- `uri` — required. Accepts `https://github.com/owner/repo`, `.../issues/N`, `.../pull/N`, or `repo://owner/repo/...`.
- `token` — optional. Falls back to `GITHUB_TOKEN` env var.

Response: a `document.ingested` event with full provenance metadata (connector, object_type, object_id, source_id, source_uri, ETag, cursor).

### Slack

Request body:

```json
{ "uri": "slack://CHANNEL_ID", "token": "xoxb-..." }
```

- `uri` — required. `slack://CHANNEL_ID` for a channel; `slack://CHANNEL_ID/TIMESTAMP` for a specific message.
- `token` — optional Bot User OAuth Token. Falls back to `SLACK_BOT_TOKEN` env var.

Required OAuth scopes: `channels:history`, `channels:read`.

Response: same `document.ingested` envelope with Slack-specific metadata (`slack_channel_id`, `slack_ts`, `slack_api_status`) and the raw Slack API JSON as content.

New connector endpoints follow the same pattern: add `request/`, `response/`, and `handler/` files and register the route in `main.go`.
