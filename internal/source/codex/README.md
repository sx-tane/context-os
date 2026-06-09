# internal/source/codex

Source connector that delegates ingestion to the [Codex CLI](https://github.com/openai/codex) plugin system.

## How it works

1. `Ingest()` and `IngestStream()` share the same `ingestWithProgress` implementation; `Ingest()` uses it in non-streaming mode.
2. `runCodex` builds the prompt string from the URI and invokes the Codex CLI via `exec` (`--sandbox read-only --ephemeral --color never -o <tmpfile>`).
3. The last agent message is written to `<tmpfile>` by Codex; stdout/stderr are captured as the run log.
4. On context cancellation the **whole process group** is killed (`SIGKILL` to `-pgid`) so child processes spawned by the Codex agent cannot keep the stdout/stderr pipes open and stall the HTTP handler.
5. The ingested content and log are returned as a single `events.Event`.

Prompts ask Codex to keep the readable source summary and append one auditable
`CONTEXTOS_LABELS_JSON:` line with entities grouped as `requirement`, `api_field`, `service`,
`dependency`, `enum`, and `db_column`, plus risks, decisions, and status. Extraction parses that
line deterministically as assistive metadata with source evidence; generic prose labels are not
accepted without provenance.

Jira prompts route through Atlassian Rovo but explicitly ask for accessible Atlassian resources
first, then Jira JQL issue search with the returned `cloudId`/URL. Generic Rovo workspace search can
return `app is not installed on this instance` even when the Jira JQL tool is usable for the
connected Atlassian account.

## Metadata keys

| Key             | Direction | Description                                                          |
| --------------- | --------- | -------------------------------------------------------------------- |
| `codex_plugin`  | in        | Required. Plugin short name such as `github`, `atlassian-rovo`, `slack`, `google-drive`, `notion`, or `sharepoint`. |
| `provider`      | out       | Set to `"codex_cli"` on successful ingestion.                        |
| `codex_prompt`  | out       | The exact prompt sent to Codex (for audit/replay).                   |
| `codex_command` | out       | The Codex executable path used.                                      |
| `codex_log`     | out       | Combined stdout/stderr from the `codex exec` run.                    |

## Supported plugins

| Constant              | Value            | Marketplace plugin              | Use for              |
| --------------------- | ---------------- | ------------------------------- | -------------------- |
| `PluginGitHub`        | `github`         | `github@openai-curated`         | GitHub repos/issues  |
| `PluginAtlassianRovo` | `atlassian-rovo` | `atlassian-rovo@openai-curated` | Jira issues/projects |
| `PluginSlack`         | `slack`          | `slack@openai-curated`          | Slack channels/DMs   |
| `PluginGoogleDrive`   | `google-drive`   | `google-drive@openai-curated`   | Drive folders/docs   |
| `PluginNotion`        | `notion`         | `notion@openai-curated`         | Notion pages/dbs     |
| `PluginSharePoint`    | `sharepoint`     | `sharepoint@openai-curated`     | SharePoint/OneDrive  |

## Prerequisites

```sh
npm install -g @openai/codex              # install CLI
codex plugin add github@openai-curated   # add GitHub plugin
codex plugin add atlassian-rovo@openai-curated # add Jira/Rovo plugin
codex plugin add slack@openai-curated    # add Slack plugin
codex login                              # local
codex login --device-auth               # remote / headless
```

Set `CODEX_BIN=/path/to/codex` when the API process cannot find the CLI on `PATH`.

## Timeout

The API handler uses a **5-minute** context timeout for Codex provider requests (vs 20 s for direct API connectors), so slower plugin-backed reads can complete without leaving requests unbounded.

## Testing

Tests use a shell-script fake for the Codex binary (`fakeCodexCommand`). The fake writes content to the `-o` output file so no real Codex installation is required.

```sh
go test ./internal/source/codex/...
```
