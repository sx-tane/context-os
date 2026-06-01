# handler/sharepoint

HTTP handlers for the `/sharepoint/*` routes.

## Handlers

| Function       | Route                         | Method | Description                                                      |
| -------------- | ----------------------------- | ------ | ---------------------------------------------------------------- |
| `Status`       | `/sharepoint/status`          | GET    | Reports whether SharePoint / Graph credentials are configured    |
| `Ingest`       | `/sharepoint/ingest`          | POST   | Fetches a SharePoint or OneDrive item and emits a raw event      |
| `IngestStream` | `/sharepoint/ingest/stream`   | POST   | Streams SharePoint Codex plugin progress as SSE then emits result |

## Request type

`request.SharePointIngest` — fields: `URI`, `Content`, `Token`, `TenantID`, `ClientID`, `ClientSecret`, `Provider`, `Metadata`.

The handler reads credentials from request fields first, then falls back to environment variables:
`SHAREPOINT_ACCESS_TOKEN`, `SHAREPOINT_TENANT_ID`, `SHAREPOINT_CLIENT_ID`, `SHAREPOINT_CLIENT_SECRET`.

Set `provider=codex` to route through the `sharepoint` Codex plugin.

## Status response

```json
{
  "connected": true,
  "access_token_configured": false,
  "client_credentials_configured": true,
  "tenant_configured": true
}
```
