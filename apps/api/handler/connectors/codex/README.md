# handler/connectors/codex

HTTP handlers for the `/codex/*` routes. Manages Codex CLI status checks,
device-auth login, and plugin re-authentication flows.

## Handlers

| Function       | Route                           | Method | Description                                                                                                                                            |
| -------------- | ------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Status`       | `/codex/status`                 | GET    | Reports CLI version, login state, and installed plugins                                                                                                |
| `Sources`      | `/codex/sources?connector=…`    | GET    | Lists readable source references through Codex for GitHub, Jira, Slack, Notion, Google Drive, and SharePoint/OneDrive                                  |
| `Login`        | `/codex/login`                  | POST   | Streams `codex login --device-auth` output as SSE log events                                                                                           |
| `PluginReauth` | `/codex/plugin-reauth?plugin=…` | POST   | Removes then re-adds a plugin with `BROWSER=echo` so the OAuth URL is printed into the SSE log instead of opening a browser on the server; streams SSE |

## Plugin names

| Query value                | Full plugin name                |
| -------------------------- | ------------------------------- |
| `github`                   | `github@openai-curated`         |
| `atlassian-rovo` or `jira` | `atlassian-rovo@openai-curated` |
| `slack`                    | `slack@openai-curated`          |
| `notion`                   | `notion@openai-curated`         |
| `googledrive`              | `google-drive@openai-curated`   |
| `sharepoint`               | `sharepoint@openai-curated`     |

## Private helpers

- `sourceDiscoveryTimeout` - bounds Codex source discovery at 5 minutes so slower plugin reads can complete while still surfacing a deterministic timeout.
- `resolveCodexBin()` — locates the Codex binary from `CODEX_BIN`, `PATH`, or common user-relative nvm paths.
- `runCodexInfo(args…)` — runs Codex with a 5-second timeout, captures combined output.
- `runCodexSSE(ctx, sw, binary, args…)` — streams a Codex sub-command to an `SSEWriter` with a 3-minute timeout.
- `runCodexSSEEnv(ctx, sw, binary, extraEnv, args…)` — like `runCodexSSE` but merges extra environment variables into the subprocess (used by `PluginReauth` to inject `BROWSER=echo`).
- `codexVersion`, `codexLoginStatus`, `codexPlugins` — parse `runCodexInfo` output into structured values. `codexLoginStatus` filters CLI warning lines so `/codex/status.account` only contains the actual Codex login status.

## Notes

`codex plugin add` has no `--device-auth` flag. `PluginReauth` sets `BROWSER=echo` in the subprocess environment so the CLI prints the OAuth URL to stdout instead of opening a browser on the server. The URL appears in the SSE log.

Codex source discovery uses the longer `sourceDiscoveryTimeout` because plugin-backed source listing can take longer than normal status checks. Timeout errors include the configured duration. GitHub discovery asks the plugin first and may use read-only `gh repo list` as fallback when `gh` is already authenticated; setup still saves only a connected source reference and does not ingest repository content. Jira discovery asks Atlassian Rovo for Jira projects through Jira JQL issue search first, since generic Rovo workspace search can be blocked by a site-install 403 even when JQL search succeeds.

> **Frontend reauth UI is not currently wired.** The `/codex/plugin-reauth` endpoint is fully functional but the frontend button has been removed. To reconnect a plugin to a different account, run in your terminal:
>
> ```
> codex plugin remove <plugin>@openai-curated && codex plugin add <plugin>@openai-curated
> ```
>
> Tracked in: **Add frontend Codex plugin re-auth flow**.
