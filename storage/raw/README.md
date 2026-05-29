# Raw Storage

Stores raw source snapshots before normalization.

Browser filesystem uploads are staged under `uploads/<upload-id>/` before ingestion. The API preserves uploaded folder relative paths there, then runs the filesystem connector against the staged file or folder so upload and path-based ingestion share the same extraction behavior.

The `uploads/` subtree is local ingest staging context. Upload IDs are implementation details and should not become stable contracts.

## Responsibilities

- Preserve original source material before downstream transformation.
- Support replay, audit, and provenance checks against raw inputs.
- Keep uploaded browser file trees intact until ingestion completes.

## Maintenance Checklist

- Document new raw storage conventions or retention rules here.
- Avoid mutating staged files after they are written.
- Keep upload-path behavior aligned with filesystem connector documentation.
