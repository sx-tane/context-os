# SharePoint Source

Placeholder source package for future SharePoint and OneDrive ingestion.

## Current State

- Exposes the connector name `sharepoint` with `files` capability.
- Uses the shared MCP connector scaffold only.
- Does not yet call Microsoft Graph or perform OAuth.

## Future Scope

- SharePoint pages, OneDrive files, Word, Excel, PowerPoint, and PDF metadata.
- Stable drive item IDs, eTags, and revision cursors.
- Microsoft Graph-backed local-first credential handling.

Keep real implementation details inside this package when issue scope starts.
