# handler/googledrive

HTTP handlers for the `/googledrive/*` routes.

## Handlers

| Function | Route                  | Method | Description                                                       |
| -------- | ---------------------- | ------ | ----------------------------------------------------------------- |
| `Status` | `/googledrive/status`  | GET    | Reports Google Drive credential-path and folder configuration     |
| `Ingest` | `/googledrive/ingest`  | POST   | Lists Docs, Sheets, and Slides in a folder and emits raw events   |

## Request type

`request.GoogleDriveIngest` — fields: `URI`, `FolderID`, `CredentialPath`, `ServiceAccountPath`, `AccessToken`, `Cursor`, and optional `Metadata`.

Credentials are read from request fields when provided, otherwise the handler falls back to `GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH`, `GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH`, `GOOGLE_DRIVE_ACCESS_TOKEN`, and `GOOGLE_DRIVE_FOLDER_ID`.
