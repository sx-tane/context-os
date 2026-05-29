# Google Drive Source

Placeholder source package for future Google Drive ingestion.

## Current State

- Exposes the connector name `googledrive` with `files` capability.
- Uses the shared MCP connector scaffold only.
- Does not yet call Google APIs or perform OAuth.

## Future Scope

- Google Docs, Sheets, Slides, and Drive file metadata.
- Stable Drive file IDs and revision cursors.
- OAuth-backed local-first credential handling.

Keep real implementation details inside this package when issue scope starts.
