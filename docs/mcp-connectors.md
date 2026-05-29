# MCP Connector Architecture

All external source integrations in ContextOS are MCP-first connectors. Each connector implements the domain `MCPSourceConnector` contract and converts source-specific input into `document.ingested` events.

## Phase 1 — local-first connectors

These connectors ingest from local paths or authenticated APIs with no hosted infrastructure dependency.

| Connector  | Source                                                                                                                          | Capability   | Issue |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------- | ------------ | ----- |
| GitHub     | Repository, issues, PRs                                                                                                         | `repository` | #7    |
| Slack      | Messages, threads, channels                                                                                                     | `messages`   | #8    |
| Jira/Rovo  | Jira issues, comments, status history, fields, and links through the Atlassian Rovo Codex plugin by default                     | `issues`     | #9    |
| Filesystem | Local files and folders, including text/code/config, OpenAPI JSON/YAML specs, spreadsheets, Word, PDF, and PowerPoint documents | `files`      | #12   |
| Google Drive | Google Docs, Sheets, Slides via OAuth 2.0 or service-account credentials | `files` | #30 |

## Phase 2 — cloud and knowledge-base connector candidates

These connectors require OAuth or API token credentials and target cloud-hosted knowledge stores. They are tracked as future scope; no Confluence source package is scaffolded right now because Atlassian context is routed through Jira/Rovo until that scope is reopened.

| Connector             | Source                                                | Capability | Issue |
| --------------------- | ----------------------------------------------------- | ---------- | ----- |
| SharePoint / OneDrive | Word, Excel, PowerPoint, PDF via Microsoft Graph      | `files`    | #31   |
| Confluence            | Deferred; use Jira/Rovo for current Atlassian context | `docs`     | #32   |
| Notion                | Pages and database entries                            | `docs`     | #33   |

## Connector output

Each connector emits raw ingestion events that are then normalized, classified, extracted, resolved, related, stored in the context graph, and analyzed for delivery mismatches.

## HTTP API surface

Connectors are exposed via the Go API (`apps/api`). Each connector has a dedicated endpoint under `/<connector>/ingest`, with filesystem also exposing `/filesystem/upload` for browser-selected files and folders.

| Method | Path                    | Connector  | Description                                                    |
| ------ | ----------------------- | ---------- | -------------------------------------------------------------- |
| GET    | `/github/status`        | GitHub     | Report configured token/account status                         |
| GET    | `/googledrive/status`   | Google Drive | Report local OAuth/service-account/folder configuration     |
| POST   | `/googledrive/ingest`   | Google Drive | Ingest Docs, Sheets, and Slides from a Drive folder         |
| POST   | `/github/ingest`        | GitHub     | Ingest a repo, issue, PR, or commit by URI                     |
| POST   | `/github/ingest/stream` | GitHub     | Stream Codex-backed GitHub ingest over SSE                     |
| GET    | `/jira/status`          | Jira       | Report Jira env configuration status                           |
| POST   | `/jira/ingest`          | Jira       | Ingest an issue or project by URI                              |
| POST   | `/jira/ingest/stream`   | Jira       | Stream Codex/Rovo-backed Jira ingest over SSE                  |
| POST   | `/filesystem/ingest`    | Filesystem | Ingest a local file or recursive folder path                   |
| POST   | `/filesystem/upload`    | Filesystem | Upload browser-selected files or folders, then ingest          |
| GET    | `/slack/status`         | Slack      | Report env/OAuth token status                                  |
| GET    | `/slack/connect`        | Slack      | Start Slack OAuth flow                                         |
| GET    | `/slack/callback`       | Slack      | Slack OAuth callback; stores token locally                     |
| POST   | `/slack/ingest`         | Slack      | Ingest a channel or message by URI                             |
| POST   | `/slack/ingest/stream`  | Slack      | Stream Codex-backed Slack ingest over SSE                      |
| GET    | `/codex/status`         | Codex      | Report CLI login and plugin status                             |
| POST   | `/codex/login`          | Codex      | Stream device-auth login output over SSE                       |
| POST   | `/codex/plugin-reauth`  | Codex      | Re-auth `github`, `atlassian-rovo`, or `slack` plugin over SSE |

Request body:

```json
{
  "uri": "https://github.com/owner/repo/issues/1",
  "token": "ghp_...",
  "provider": "token"
}
```

- `uri` — required. Accepts `https://github.com/owner/repo`, `.../issues/N`, `.../pull/N`, or `repo://owner/repo/...`.
- `token` — optional. Falls back to `GITHUB_TOKEN` env var.
- `provider` — optional. Omit or set `token` for direct GitHub API ingest. Set `codex` to delegate to the Codex CLI GitHub plugin; use `/github/ingest/stream` for live progress.

Response: a `document.ingested` event with full provenance metadata (connector, object_type, object_id, source_id, source_uri, ETag, cursor).

### Slack

Request body:

```json
{ "uri": "slack://CHANNEL_ID", "token": "xoxb-...", "provider": "token" }
```

- `uri` — required. `slack://CHANNEL_ID` for a channel; `slack://CHANNEL_ID/TIMESTAMP` for a specific message.
- `token` — optional Bot User OAuth Token. Falls back to saved OAuth token, then `SLACK_BOT_TOKEN` env var.
- `provider` — optional. Omit or set `token` for direct Slack API ingest. Set `codex` to delegate to the Codex CLI Slack plugin; use `/slack/ingest/stream` for live progress.

Required OAuth scopes: `channels:history`, `channels:read`.

Response: same `document.ingested` envelope with Slack-specific metadata (`slack_channel_id`, `slack_ts`, `slack_api_status`) and the raw Slack API JSON as content.

### Jira

Request body:

```json
{
  "uri": "https://site.atlassian.net/browse/PROJ-123",
  "token": "ATATT...",
  "email": "name@example.com",
  "api_base_url": "https://site.atlassian.net",
  "provider": "token"
}
```

- `uri` — required unless inline `content` is provided. Accepts Jira browse URLs, `jira://issue/PROJ-123`, and `jira://project/PROJ`.
- `token`, `email`, and `api_base_url` — optional request overrides. The API falls back to `JIRA_TOKEN`, `JIRA_EMAIL`, and `JIRA_BASE_URL`.
- `provider` — set `codex` to delegate to `atlassian-rovo@openai-curated`; use `/jira/ingest/stream` for live progress. Token/env mode remains available for direct Jira REST checks.
- `metadata` — optional string map passed through to the connector for replay and audit hints.

Response: the common `document.ingested` envelope with Jira keys such as `jira_issue_key`, `jira_project_key`, `jira_host`, `jira_api_status`, and `jira_updated`.

### Filesystem

Path request body:

```json
{
  "uri": "docs/"
}
```

- `uri` — accepts a local file or directory path, `file://` URI, or `filesystem://` URI.
- `include` and `exclude` — optional advanced glob-style path rules evaluated before each file is read.
- `content` — optional inline content for small fixtures; otherwise the connector reads the local file.
- `metadata.filesystem_max_files` — optional folder guardrail; defaults to `1000`.
- `metadata.filesystem_max_file_size` — optional per-file byte guardrail; defaults to `10485760`.

Browser upload request:

```http
POST /filesystem/upload
Content-Type: multipart/form-data
```

- `files` — one or more uploaded file parts.
- `paths` — matching browser relative paths, using `file.webkitRelativePath` for folder uploads and `file.name` for normal file uploads.

Uploads are staged under `storage/raw/uploads/<upload-id>/` before ingestion, so users can choose files or folders outside the repository while the connector still processes local files with stable replay metadata. Server path ingest remains available for developer workflows where the API can already see the path.

Supported file extraction:

| Format            | Extensions                                      | Extraction                                                                       |
| ----------------- | ----------------------------------------------- | -------------------------------------------------------------------------------- |
| Folder            | Directory path                                  | Recursive child-file events with stable per-file IDs                             |
| Text and Markdown | `.txt`, `.md`                                   | Read directly                                                                    |
| Code and config   | `.go`, `.ts`, `.json`, `.yaml`, `.toml`, `.sql` | Read directly; OpenAPI JSON/YAML receives endpoint/schema metadata when detected |
| Spreadsheet       | `.xlsx`, `.csv`                                 | Cell, sheet, row, value, and formula facts                                       |
| Word document     | `.docx`                                         | Paragraph text                                                                   |
| PDF               | `.pdf`                                          | Best-effort page text                                                            |
| PowerPoint        | `.pptx`                                         | Slide text                                                                       |

Response metadata includes original path, extension, extracted format, content hash, modified time, size, spreadsheet summary fields including formula count when applicable, OpenAPI summary fields when a JSON/YAML file is an API spec, and `filesystem_upload_*` keys for browser uploads. Folder responses preserve the existing first-event fields and add `events`, `previews`, `metadata_items`, and `event_count`; each child event has `filesystem_ingest_mode=folder`, `filesystem_root`, `filesystem_relative_path`, emitted file count, skipped count, and first skipped path metadata.

New connector endpoints follow the same pattern: add `request/`, `response/`, and `handler/` files and register the route in `main.go`.

### Google Drive

Request body:

```json
{
  "uri": "https://drive.google.com/drive/folders/1234567890",
  "credential_path": "/Users/name/.config/context-os/google-authorized-user.json"
}
```

- `uri` or `folder_id` selects the Drive folder to scan. `GOOGLE_DRIVE_FOLDER_ID` is the local fallback when the request omits both.
- `credential_path` points to a local Google `authorized_user` OAuth JSON file containing `client_id`, `client_secret`, and `refresh_token`.
- `service_account_path` points to a local Google `service_account` JSON file. Either credential path is enough.
- `access_token` is an optional already-issued bearer token override for local automation.
- `cursor` is an optional RFC3339 modified-time watermark; only files newer than the cursor are listed.

Response metadata includes the stable Drive file ID, modified time, export format, credential type, and `url=https://drive.google.com/file/d/<file-id>/view`. Replay of an unchanged file reuses the same event ID because the connector hashes file ID and `modifiedTime`.
