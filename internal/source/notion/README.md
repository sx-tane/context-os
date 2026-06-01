# Notion Source

MCP source connector for Notion pages and databases via the Notion API v1.

## URI formats

| Format | Resolves to |
|--------|-------------|
| `notion://page/{id}` | Fetches a Notion page block tree |
| `notion://database/{id}` | Fetches a Notion database |
| `https://www.notion.so/{title}-{32-char-id}` | Page resolved from URL |

## Authentication

The connector resolves a token in this priority order:

1. `notion_token` metadata key on the ingest request
2. `NOTION_TOKEN` environment variable

All requests send `Notion-Version: 2022-06-28`.

## Metadata enriched

| Key | Source |
|-----|--------|
| `notion_page_id` | Parsed from URI |
| `notion_database_id` | Parsed from URI |
| `notion_last_edited_time` | API response |
| `notion_url` | API response |
| `notion_title` | API response |
| `object_type` | `page` or `database` |
| `object_id` | Canonical object ID |

## Idempotency

The cursor is set to the `last_edited_time` from the Notion API response, making repeated ingests of unchanged content replay-safe.

## Exported symbols

- `NewConnector() contracts.MCPSourceConnector` — production connector using `https://api.notion.com/v1`
- `NewConnectorWithOptions(apiBaseURL string, client HTTPClient) contracts.MCPSourceConnector` — for tests
- `MetadataToken = "notion_token"` — metadata key for the integration token
- `HTTPClient` — interface satisfied by `*http.Client`
