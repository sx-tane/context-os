# internal/source/codex

Source connector that delegates ingestion to the [Codex CLI](https://github.com/openai/codex) plugin system.

## How it works

1. `Ingest()` and `IngestStream()` share the same `ingestWithProgress` implementation; `Ingest()` uses it in non-streaming mode.
2. `runCodex` builds the prompt string from the URI and invokes the Codex CLI via `exec` (`--sandbox read-only --ephemeral --color never -o <tmpfile>`).
3. The last agent message is written to `<tmpfile>` by Codex; stdout/stderr are captured as the run log.
4. On context cancellation the **whole process group** is killed (`SIGKILL` to `-pgid`) so child processes spawned by the Codex agent cannot keep the stdout/stderr pipes open and stall the HTTP handler.
5. The ingested content and log are returned as a single `events.Event`.

## Metadata keys

| Key             | Direction | Description                                                          |
| --------------- | --------- | -------------------------------------------------------------------- |
| `codex_plugin`  | in        | Required. Plugin short name: `github`, `atlassian-rovo`, or `slack`. |
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

## Prerequisites

```sh
npm install -g @openai/codex              # install CLI
codex plugin add github@openai-curated   # add GitHub plugin
codex plugin add atlassian-rovo@openai-curated # add Jira/Rovo plugin
codex plugin add slack@openai-curated    # add Slack plugin
codex login                              # local
codex login --device-auth               # remote / headless
```

`start-all.sh` automates steps 1–3. Step 4 must be done by the user once.

## Timeout

The API handler uses a **120-second** context timeout for Codex provider requests (vs 20 s for direct API connectors). The Vite dev proxy is configured with a matching 3-minute socket timeout.

## Testing

Tests use a shell-script fake for the Codex binary (`fakeCodexCommand`). The fake writes content to the `-o` output file so no real Codex installation is required.

```sh
go test ./internal/source/codex/...
```
