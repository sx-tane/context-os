# connectors

Svelte components that render individual source-connector forms and identity badges.

## Components

All ingesting connector components read the active workspace from `$lib/projectStore` and pass `$project.workspacePath` to `runConnectorIngest` as `workspace_id`. This keeps direct and Codex-backed ingest persistence scoped to the workspace selected in the frontend.

### ConnectorCard

Generic layout shell for any connector section.

| Prop          | Type       | Purpose                                               |
| ------------- | ---------- | ----------------------------------------------------- |
| `title`       | `string`   | Section heading (shown in small-caps).                |
| `description` | `string`   | Hint text rendered below the heading.                 |
| `examples`    | `string[]` | Optional URI example chips rendered as `<code>` tags. |

Exposes a `<slot />` where the connector puts its form controls. All connector components wrap themselves in `ConnectorCard`.

---

### SourceConnector

Generic connector form for `DirectSourceConnectorKind` connectors (currently filesystem). Driven entirely by props derived from `SourceConnectorConfig`.

| Prop                                 | Type                        | Purpose                                             |
| ------------------------------------ | --------------------------- | --------------------------------------------------- |
| `connector`                          | `DirectSourceConnectorKind` | Which connector to call.                            |
| `title`, `description`, `examples`   | `string` / `string[]`       | Forwarded to `ConnectorCard`.                       |
| `defaultUri`                         | `string`                    | Pre-filled URI value.                               |
| `uriLabel`, `uriPlaceholder`         | `string`                    | Label and placeholder for the URI field.            |
| `submitLabel`                        | `string`                    | Label on the server-path submit button.             |
| `uploadEnabled`                      | `boolean`                   | Shows the file/folder upload control when `true`.   |
| `tokenLabel`, `tokenPlaceholder`     | `string`                    | Optional token field; hidden when labels are empty. |
| `contentLabel`, `contentPlaceholder` | `string`                    | Optional inline-content field.                      |
| `metadataFields`                     | `SourceMetadataField[]`     | Extra key/value fields appended to ingest metadata. |
| `supportedFormats`                   | `SupportedFormat[]`         | Shown in a collapsible "Supported formats" panel.   |

Internally calls `runConnectorIngest` for server-path ingest and `postFilesystemUpload` directly for file/folder uploads. Manages its own `AbortController` and cleans up on `onDestroy`.

Server-path ingest forwards the active workspace path through `workspace_id`; file and folder uploads use the upload endpoint directly.

---

### GitHubConnector

Connector form for GitHub repository/issue ingestion via the Codex GitHub plugin or a direct personal access token.

- **Codex mode**: streams SSE through `streamCodexIngest`; shows a live log via `LogPanel`.
- **Token mode**: calls `postIngest` with the provided token.
- Both modes include the active workspace path in the ingest request.
- URI format: `github://<owner>/<repo>` or an issue/PR URL.

---

### JiraConnector

Connector form for Jira/Rovo ingestion via the Atlassian Rovo Codex plugin or direct API token.

- **Codex (Rovo) mode**: SSE stream; plugin must be installed and enabled.
- **Token mode**: `postIngest` with `JIRA_BASE_URL` / `JIRA_EMAIL` / `JIRA_API_TOKEN` sourced from env or the token field.
- Both modes include the active workspace path in the ingest request.
- URI format: `jira://<host>/issue/<KEY>` or a JQL query string.

---

### SlackConnector

Connector form for Slack channel/message ingestion via the Codex Slack plugin or a direct bot token.

- **Codex mode**: SSE stream.
- **Token mode**: `postIngest` with the bot token.
- Both modes include the active workspace path in the ingest request.
- URI format: `slack://<workspace>/<channel-id>` or a Slack message permalink.

---

### NotionConnector

Connector form for Notion page/database ingestion via the Notion Codex plugin or a direct integration token.

- **Codex mode**: SSE stream; plugin must be installed and enabled.
- **Token mode**: `postIngest` with `NOTION_TOKEN` sourced from env or the token field.
- Both modes include the active workspace path in the ingest request.
- URI formats: `notion://page/<id>`, `notion://database/<id>`, or a `notion.so` URL.

---

### SharePointConnector

Connector form for SharePoint and OneDrive file ingestion via the SharePoint Codex plugin, a direct access token, or OAuth2 client credentials.

- **Codex mode**: SSE stream; plugin must be installed and enabled.
- **Token/credentials mode**: `postIngest` with access token OR tenant/client/secret fields.
- Both modes include the active workspace path in the ingest request.
- URI formats: `sharepoint://sites/<siteId>/items/<itemId>`, `sharepoint://drives/<driveId>/items/<itemId>`.

---

### CodexBadge

Inline status badge for a Codex plugin. Displays `installed` / `enabled` state with coloured dots. Used inside the Codex-backed connector forms to surface plugin health at a glance.

| Prop        | Type         | Purpose                                        |
| ----------- | ------------ | ---------------------------------------------- |
| `plugin`    | `string`     | Plugin display name.                           |
| `installed` | `boolean`    | Whether Codex reports the plugin as installed. |
| `enabled`   | `boolean`    | Whether the plugin is currently enabled.       |
| `onReauth`  | `() => void` | Optional callback wired to the re-auth button. |
