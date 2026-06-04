# API Request Types

This folder defines HTTP request payload contracts consumed by API handlers.

## Responsibilities

- Keep request structs explicit and small.
- Align JSON tags with handler decoding expectations.
- Avoid embedding stage-internal implementation details.

## Current Files

- `chat.go`: `ChatQuery` request body for `/chat/query` and `/chat/query/stream`, including workspace scope, connector/source hints, timezone, local date, response-language hint, and result limit.
- `ingest.go`: request shapes used by ingest endpoints — `GithubIngest`, `SlackIngest`, `JiraIngest`, `GoogleDriveIngest`, `NotionIngest`, `SharePointIngest`, and `FilesystemIngest`.
- `presentation.go`: `PresentationFindings` request body for `/presentation/findings`, including workspace scope, connector input, role, provider, force refresh, and execution toggle.

## Request Types

| Type                | Endpoint                   | Key fields                                                                              |
| ------------------- | -------------------------- | --------------------------------------------------------------------------------------- |
| `GithubIngest`      | `POST /github/ingest`      | `uri`, `token`, `provider`                                                              |
| `SlackIngest`       | `POST /slack/ingest`       | `uri`, `token`, `provider`                                                              |
| `JiraIngest`        | `POST /jira/ingest`        | `uri`, `token`, `email`, `api_base_url`, `provider`, `cursor`                           |
| `GoogleDriveIngest` | `POST /googledrive/ingest` | `uri`, `folder_id`, `credential_path`, `service_account_path`, `access_token`, `cursor` |
| `NotionIngest`      | `POST /notion/ingest`      | `uri`, `token`, `provider`, `cursor`, `metadata`                                        |
| `SharePointIngest`  | `POST /sharepoint/ingest`  | `uri`, `access_token`, client credential fields, `provider`, `metadata`                 |
| `FilesystemIngest`  | `POST /filesystem/ingest`  | `uri`, `content`, `cursor`, `include`, `exclude`                                        |
| `ChatQuery`         | `POST /chat/query` and `POST /chat/query/stream` | `workspace_id`, `workspace_path`, `message`, `connector`, `source_uri`, `timezone`, `local_date`, `response_language`, `limit` |
| `PresentationFindings` | `POST /presentation/findings` | `workspace_id`, `connector`, `uri`, `content`, `cursor`, `provider`, `role`, `include_execution`, `force_refresh`, `metadata` |

## Maintenance Checklist

- Update request structs when endpoint inputs change.
- Regenerate swagger docs if request contracts are modified.
- Add or update handler tests when validation behavior changes.
