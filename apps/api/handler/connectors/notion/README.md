# handler/connectors/notion

HTTP handlers for the `/notion/*` routes.

## Handlers

| Function       | Route                    | Method | Description                                                  |
| -------------- | ------------------------ | ------ | ------------------------------------------------------------ |
| `Status`       | `/notion/status`         | GET    | Reports whether a Notion integration token is configured     |
| `Ingest`       | `/notion/ingest`         | POST   | Fetches a Notion page or database and emits a raw event      |
| `IngestStream` | `/notion/ingest/stream`  | POST   | Streams Notion Codex plugin progress as SSE then emits result |

## Request type

`request.NotionIngest` — fields: `URI`, `Content`, `Token`, `Provider`, `Metadata`.

Token is read from the request field when provided; otherwise falls back to the `NOTION_TOKEN` environment variable.
Set `provider=codex` to route through the `notion` Codex plugin.

## Status response

```json
{
  "connected": true,
  "token_configured": true
}
```
