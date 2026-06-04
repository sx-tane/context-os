# Google Drive Source

Google Drive MCP connector for Docs, Sheets, and Slides folder ingestion.

## Current Behavior

- Supports Google OAuth authorized-user credential files via `GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH` or request metadata `googledrive_oauth_credentials_path`.
- Supports Google service-account credential files via `GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH` or request metadata `googledrive_service_account_path`.
- Optional direct access-token override via `GOOGLE_DRIVE_ACCESS_TOKEN` or request metadata `googledrive_access_token`.
- Lists supported files from a Drive folder URI (`https://drive.google.com/drive/folders/<id>` or `googledrive://folder/<id>`).
- Exports Docs as plain text, Sheets as CSV parsed into tabular text, and Slides as per-slide text.
- Emits one `document.ingested` event per file with `url`, `googledrive_file_id`, `googledrive_mime_type`, and `googledrive_modified_time` metadata.
- Uses a stable event ID derived from Drive file ID and `modifiedTime`, so replaying an unchanged file reuses the same event identity.

## Replay And Provenance

- `source_id=googledrive:file:<file-id>` keeps the upstream file identity stable across replays.
- `event_id` hashes `<file-id> + <modifiedTime>` so a changed file produces a new event while an unchanged file does not.
- `Metadata["url"]` stores `https://drive.google.com/file/d/<file-id>/view`.
- Each file event uses its Drive `modifiedTime` as `source_cursor`.

## Operational Notes

- Folder discovery may also fall back to `GOOGLE_DRIVE_FOLDER_ID` when the request omits a URI.
- Rate-limit (`429`) and server (`5xx`) responses back off before surfacing a retryable connector error.
- Credentials stay on the local filesystem and are never written into emitted source artifacts.
