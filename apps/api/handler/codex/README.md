# handler/codex

HTTP handlers for the `/codex/*` routes. Manages Codex CLI status checks,
device-auth login, and plugin re-authentication flows.

## Handlers

| Function       | Route                           | Method | Description                                                                                                                                            |
| -------------- | ------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Status`       | `/codex/status`                 | GET    | Reports CLI version, login state, and installed plugins                                                                                                |
| `Login`        | `/codex/login`                  | POST   | Streams `codex login --device-auth` output as SSE log events                                                                                           |
| `PluginReauth` | `/codex/plugin-reauth?plugin=…` | POST   | Removes then re-adds a plugin with `BROWSER=echo` so the OAuth URL is printed into the SSE log instead of opening a browser on the server; streams SSE |

## Plugin names

| Query value                | Full plugin name                |
| -------------------------- | ------------------------------- |
| `github`                   | `github@openai-curated`         |
| `atlassian-rovo` or `jira` | `atlassian-rovo@openai-curated` |
| `slack`                    | `slack@openai-curated`          |

## Private helpers

- `resolveCodexBin()` — locates the Codex binary, falling back to nvm paths.
- `runCodexInfo(args…)` — runs Codex with a 5-second timeout, captures combined output.
- `runCodexSSE(ctx, sw, binary, args…)` — streams a Codex sub-command to an `SSEWriter` with a 3-minute timeout.
- `runCodexSSEEnv(ctx, sw, binary, extraEnv, args…)` — like `runCodexSSE` but merges extra environment variables into the subprocess (used by `PluginReauth` to inject `BROWSER=echo`).
- `codexVersion`, `codexLoginStatus`, `codexPlugins` — parse `runCodexInfo` output into structured values.

## Notes

`codex plugin add` has no `--device-auth` flag. `PluginReauth` sets `BROWSER=echo` in the subprocess environment so the CLI prints the OAuth URL to stdout instead of opening a browser on the server. The URL appears in the SSE log.

> **Frontend reauth UI is not currently wired.** The `/codex/plugin-reauth` endpoint is fully functional but the frontend button has been removed. To reconnect a plugin to a different account, run in your terminal:
>
> ```
> codex plugin remove <plugin>@openai-curated && codex plugin add <plugin>@openai-curated
> ```
>
> Tracked in: **Add frontend Codex plugin re-auth flow**.
