# Notion Source

Placeholder source package for future Notion ingestion.

## Current State

- Exposes the connector name `notion` with `docs` capability.
- Uses the shared MCP connector scaffold only.
- Does not yet call Notion APIs or perform OAuth.

## Future Scope

- Pages, databases, comments, and block trees.
- Stable page/database IDs and edit cursors.
- Metadata that keeps Notion evidence traceable without leaking Notion payload shapes downstream.

Keep real implementation details inside this package when issue scope starts.
