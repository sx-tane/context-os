# SharePoint Source

MCP source connector for SharePoint and OneDrive files via Microsoft Graph API v1.0.

## URI formats

| Format | Resolves to |
|--------|-------------|
| `sharepoint://sites/{siteId}/items/{itemId}` | SharePoint site drive item |
| `sharepoint://drives/{driveId}/items/{itemId}` | OneDrive drive item |
| `https://graph.microsoft.com/v1.0/...` | Raw Graph API URL |

## Authentication

The connector resolves credentials in this priority order:

1. `sharepoint_access_token` metadata key
2. `SHAREPOINT_ACCESS_TOKEN` environment variable
3. OAuth2 client credentials (metadata keys `sharepoint_tenant_id`, `sharepoint_client_id`, `sharepoint_client_secret`)
4. `SHAREPOINT_TENANT_ID` + `SHAREPOINT_CLIENT_ID` + `SHAREPOINT_CLIENT_SECRET` environment variables

## Metadata enriched

| Key | Source |
|-----|--------|
| `sharepoint_site_id` | Parsed from URI / API |
| `sharepoint_item_id` | Parsed from URI / API |
| `sharepoint_item_name` | API response |
| `sharepoint_mime_type` | API response |
| `sharepoint_etag` | API response |
| `sharepoint_modified_time` | API response |
| `sharepoint_drive_id` | API response |

## Idempotency

The cursor is set to the eTag from the Graph API response. Repeated ingests of unchanged content are replay-safe.

## Exported symbols

- `NewConnector() contracts.MCPSourceConnector` — production connector using Graph and login.microsoftonline.com
- `NewConnectorWithOptions(graphAPIBase, tokenBase string, client HTTPClient) contracts.MCPSourceConnector` — for tests
- `MetadataAccessToken`, `MetadataTenantID`, `MetadataClientID`, `MetadataClientSecret` — metadata key constants
- `HTTPClient` — interface satisfied by `*http.Client`
