# handler/slack

HTTP handlers for the `/slack/*` routes, including OAuth 2.0 flow and
local token persistence.

## Handlers

| Function       | Route                  | Method | Description                                                      |
| -------------- | ---------------------- | ------ | ---------------------------------------------------------------- |
| `Status`       | `/slack/status`        | GET    | Reports token availability (env/oauth/none) and OAuth readiness  |
| `Connect`      | `/slack/connect`       | GET    | Generates CSRF state and redirects to Slack OAuth consent page   |
| `Callback`     | `/slack/callback`      | GET    | Verifies state, exchanges code for token, persists it locally    |
| `Ingest`       | `/slack/ingest`        | POST   | Ingests a Slack channel or message via native or Codex connector |
| `IngestStream` | `/slack/ingest/stream` | POST   | Streams Codex Slack plugin progress as SSE, then emits result    |

## OAuth flow

1. `SLACK_CLIENT_ID` and `SLACK_CLIENT_SECRET` must be set.
2. `GET /slack/connect` → redirects to Slack.
3. Slack redirects to `GET /slack/callback?code=…&state=…`.
4. Token is stored at `$XDG_CONFIG_HOME/context-os/slack-token.json` (mode 0600).
5. `GET /slack/status` returns `source: "oauth"` once the token is saved.

`SLACK_REDIRECT_URI` overrides the default `http://localhost:8080/slack/callback`.

## Private helpers

- `generateOAuthState`, `purgeExpiredOAuthStates` — CSRF token lifecycle.
- `exchangeCode` — calls `oauth.v2.access` to exchange authorization code.
- `saveToken`, `loadToken`, `tokenPath` — local token persistence.
- `writeOAuthPage` — self-closing HTML result page that sends a `postMessage` to the opener.
