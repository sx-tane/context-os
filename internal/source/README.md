# Source Domain

The source domain converts external systems into ContextOS ingestion events. It is the boundary between tool-specific data and the local-first pipeline.

## Responsibility

- Represent each external source as an `MCPSourceConnector`.
- Validate source requests before producing events.
- Attach connector metadata and provenance.
- Keep connector behavior replay-safe as integrations become real API clients.

## Input And Output

```mermaid
flowchart LR
  request[contracts.SourceRequest]
  connector[MCPConnector]
  event[events.Event document.ingested]

  request --> connector --> event
```

## Core Implementation

`MCPConnector` is the shared current implementation used by all source packages while real API adapters are developed.

```go
type MCPConnector struct {
    name         string
    capabilities []contracts.Capability
}

func NewMCPConnector(name string, capabilities ...contracts.Capability) MCPConnector
func (c MCPConnector) Name() string
func (c MCPConnector) Capabilities() []contracts.Capability
func (c MCPConnector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error)
```

## Behavior

- Respects context cancellation.
- Rejects requests where both `Content` and `URI` are blank.
- Creates metadata with `connector` and `mcp` values.
- Copies `URI` to `source_uri` and `Cursor` to `source_cursor` when present.
- Copies request metadata into the emitted event metadata.
- Uses `URI` as the event subject when present, otherwise uses the connector name.
- Emits a single `document.ingested` event.
- Returns structured `contracts.ConnectorError` values for cancellation and validation failures.

### Cursor Meaning

`Cursor` is connector replay state. It is close to a snapshot marker, but it should not be treated as the full source snapshot or document body. The source content still comes from `Content`, `URI`, or the real API call. The cursor only records the source position used for incremental reads and safe retries.

Practical examples:

- GitHub: the last API page cursor or latest processed event/issue update marker.
- Slack: the last message timestamp read from a channel or thread.
- Jira: the latest processed issue update timestamp or search pagination token.
- Filesystem: a file version, modification watermark, or per-file content hash used to detect whether rereading is needed.

When the same `SourceRequest` is replayed with the same URI, content, cursor, and stable metadata, the connector should produce the same logical ingestion event. When a source has no cursor yet, keep it empty rather than inventing unstable values.

## Connector Wrappers

| Package                                   | Name          | Capability   |
| ----------------------------------------- | ------------- | ------------ |
| [codex](codex/codex.go)                   | `codex-cli`   | plugin-based |
| [github](github/github.go)                | `github`      | `repository` |
| [slack](slack/slack.go)                   | `slack`       | `messages`   |
| [jira](jira/jira.go)                      | `jira`        | `issues`     |
| [filesystem](filesystem/filesystem.go)    | `filesystem`  | `files`      |
| [googledrive](googledrive/googledrive.go) | `googledrive` | `files`      |
| [notion](notion/notion.go)                | `notion`      | `docs`       |
| [sharepoint](sharepoint/sharepoint.go)    | `sharepoint`  | `files`      |

Each wrapper currently exposes:

```go
func NewConnector() contracts.MCPSourceConnector
```

The Codex wrapper is a provider connector used by the API when a request sets `provider=codex`. It delegates GitHub, Jira, or Slack ingestion to installed Codex CLI plugins (`github@openai-curated`, `atlassian-rovo@openai-curated`, `slack@openai-curated`) and preserves the prompt, command path, and run log in event metadata for audit and replay.

Spreadsheet and OpenAPI extraction now live inside the [filesystem](filesystem/filesystem.go) connector package as file-type handlers. They enter through `filesystem` so synced local folders stay the single file-ingestion path. Filesystem is the parent file, and child files own file-type families: `text.go`, `spreadsheet.go`, `openapi.go`, `office.go`, and `pdf.go`.

There is no Confluence source package scaffold right now. Atlassian context routes through Jira/Rovo until issue #32 is explicitly reopened as a separate source implementation.

### GitHub Connector Metadata Mapping

The GitHub connector enriches ingestion metadata from common GitHub URIs (`repo://`, `github://`, `https://github.com`, and `https://api.github.com/repos/...`).

- repository artifact
  - `object_type=repository`
  - `object_id=<owner>/<repo>`
  - `source_id=github:repository:<owner>/<repo>`
- issue artifact
  - `object_type=issue`
  - `object_id=<owner>/<repo>#<number>`
  - `source_id=github:issue:<owner>/<repo>#<number>`
- pull request artifact
  - `object_type=pull_request`
  - `object_id=<owner>/<repo>#<number>`
  - `source_id=github:pull_request:<owner>/<repo>#<number>`

Additional enriched metadata keys:

- `github_owner`
- `github_repo`
- `github_number` (issue/PR only)

If request metadata already provides `object_type`, `object_id`, or `source_id`, the connector preserves those explicit values.

### Jira Connector Metadata Mapping

The Jira connector enriches issue and project URIs from Jira browse URLs, `jira://issue/<KEY>`, and `jira://project/<KEY>`.

- issue artifact
  - `object_type=issue`
  - `object_id=<ISSUE-KEY>`
  - `source_id=jira:issue:<ISSUE-KEY>`
- project artifact
  - `object_type=project`
  - `object_id=<PROJECT-KEY>`
  - `source_id=jira:project:<PROJECT-KEY>`

Additional enriched metadata keys:

- `jira_issue_key`
- `jira_project_key`
- `jira_host`
- `jira_api_base_url`
- `jira_api_status`
- `jira_updated`

Request metadata or API fields may provide `jira_token`, `jira_email`, `jira_api_base_url`, and `jira_expand`. If omitted, the connector falls back to `JIRA_TOKEN`, `JIRA_EMAIL`, and `JIRA_BASE_URL` where applicable.

### Filesystem Connector Metadata Mapping

The filesystem connector reads local files or folders, applies explicit include/exclude rules, extracts common document formats, parses spreadsheet formats directly, and annotates OpenAPI JSON/YAML specs with endpoint and schema metadata. A directory URI is walked recursively in deterministic order and emits one `document.ingested` event per supported child file.

- `object_type=file`
- `object_id=<local-path>`
- `source_id=filesystem:file:<local-path>`
- `path`
- `filesystem_format`
- `filesystem_extension`
- `filesystem_content_hash`
- `filesystem_modified_at`
- `filesystem_size`
- `filesystem_include`
- `filesystem_exclude`
- `filesystem_ingest_mode`
- `filesystem_root`
- `filesystem_relative_path`
- `filesystem_folder_file_count`
- `filesystem_folder_skipped_count`
- `filesystem_folder_first_error`
- `filesystem_max_files`
- `filesystem_max_file_size`
- `filesystem_spreadsheet_sheets`
- `filesystem_spreadsheet_cells`
- `filesystem_spreadsheet_rows`
- `filesystem_spreadsheet_formulas`
- `openapi_title`
- `openapi_version`
- `openapi_spec_version`
- `openapi_content_hash`
- `openapi_path_count`
- `openapi_operation_count`
- `openapi_schema_count`
- `openapi_enum_count`
- `openapi_endpoint_pointers`
- `openapi_schema_pointers`
- `openapi_enum_pointers`
- `openapi_source_locations`

The content hash is used as the replay cursor when no cursor is provided. Folder child events keep per-file object IDs and `source_id` values, while folder metadata records the root, relative path, total emitted file count, skipped count, and first skipped path. Supported extraction paths include text/code/config files, OpenAPI JSON/YAML specs, `.csv`, `.xlsx`, `.docx`, `.pptx`, and best-effort `.pdf` text extraction.

## Dependencies

```mermaid
flowchart TD
  source[internal/source]
  contracts[domain/contracts]
  events[domain/events]
  ingestion[internal/ingestion]

  source --> contracts
  source --> events
  ingestion --> source
```

## Example Usage

```go
pipe := ingestion.NewPipeline(githubsource.NewConnector())
result, err := pipeline.Run(ctx, pipe, contracts.SourceRequest{
    URI:     "repo://example",
    Content: "presentation layer expects refundStatus but service layer has missingRefundState mismatch",
})
```

## Implementation Notes

- When a connector becomes a real API adapter, preserve the `MCPSourceConnector` contract and keep source-specific parsing inside the connector package.
- Use stable upstream IDs in metadata to support idempotency and replay checks.
- Use `object_type` and `object_id` metadata when connector errors need source artifact provenance.
- Do not let source packages import downstream stages. They should only emit events.
- For large payloads, metadata should point to raw storage while `Content` carries the processing text or summary needed by the next stage.

## Production Requirements

- Each connector must expose stable external artifact IDs and cursor/checkpoint metadata.
- Re-ingesting the same upstream artifact must produce the same logical event identity.
- Connector errors must include connector name, source URI, and retryability.
- Connector output must preserve enough provenance for downstream evidence bundles.
