# Raw Storage

Stores raw source snapshots before normalization.

Browser filesystem uploads are staged under `uploads/<upload-id>/` before ingestion. The API preserves uploaded folder relative paths there, then runs the filesystem connector against the staged file or folder so upload and path-based ingestion share the same extraction behavior.
