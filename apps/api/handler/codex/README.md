# handler/codex

HTTP handlers for the `/codex/*` routes. Manages Codex CLI status checks,
device-auth login, and plugin re-authentication flows.

## Handlers

| Function       | Route                           | Method | Description                                                    |
| -------------- | ------------------------------- | ------ | -------------------------------------------------------------- |
| `Status`       | `/codex/status`                 | GET    | Reports CLI version, login state, and installed plugins        |
| `Login`        | `/codex/login`                  | POST   | Streams `codex login --device-auth` output as SSE log events   |
| `PluginReauth` | `/codex/plugin-reauth?plugin=…` | POST   | Removes and re-adds a plugin to force fresh OAuth; streams SSE |

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
- `codexVersion`, `codexLoginStatus`, `codexPlugins` — parse `runCodexInfo` output into structured values.
