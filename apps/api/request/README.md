# API Request Types

This folder defines HTTP request payload contracts consumed by API handlers.

## Responsibilities

- Keep request structs explicit and small.
- Align JSON tags with handler decoding expectations.
- Avoid embedding stage-internal implementation details.

## Current Files

- `ingest.go`: request shapes used by ingest endpoints — `GithubIngest`, `SlackIngest`, `JiraIngest`, `GoogleDriveIngest`, and `FilesystemIngest`.

## Request Types

| Type | Endpoint | Key fields |
| ---- | -------- | ---------- |
| `GithubIngest` | `POST /github/ingest` | `uri`, `token`, `provider` |
| `SlackIngest` | `POST /slack/ingest` | `uri`, `token`, `provider` |
| `JiraIngest` | `POST /jira/ingest` | `uri`, `token`, `email`, `api_base_url`, `provider`, `cursor` |
| `GoogleDriveIngest` | `POST /googledrive/ingest` | `uri`, `folder_id`, `credential_path`, `service_account_path`, `access_token`, `cursor` |
| `FilesystemIngest` | `POST /filesystem/ingest` | `uri`, `content`, `cursor`, `include`, `exclude` |

## Maintenance Checklist

- Update request structs when endpoint inputs change.
- Regenerate swagger docs if request contracts are modified.
- Add or update handler tests when validation behavior changes.
