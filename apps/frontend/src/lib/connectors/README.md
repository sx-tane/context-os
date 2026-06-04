# connectors

Static connector configuration consumed by source setup and connector routes.

## Responsibilities

- Define direct-source connector labels, example URIs, metadata fields, and upload capabilities.
- Keep connector setup copy consistent with backend endpoint behavior.
- Avoid network logic; connector API calls belong in `$lib/api` or ingest runners.

## Files

| File | Purpose |
| --- | --- |
| `sourceConnectorConfigs.ts` | Configuration records for filesystem and direct connector setup UI. |

## Maintenance Notes

- Update component docs under `components/connectors/` when config changes affect rendered controls.
- Keep supported formats and examples aligned with API handler README files.
