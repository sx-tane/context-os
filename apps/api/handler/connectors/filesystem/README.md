# handler/connectors/filesystem

HTTP handlers for the `/filesystem/*` routes.

## Handlers

| Function | File                   | Route                | Method | Description                                                        |
| -------- | ---------------------- | -------------------- | ------ | ------------------------------------------------------------------ |
| `Ingest` | `filesystem.go`        | `/filesystem/ingest` | POST   | Ingests a local file or folder by path                             |
| `Upload` | `filesystem_upload.go` | `/filesystem/upload` | POST   | Stages multipart uploads then ingests through filesystem connector |

## Ingest

Accepts `request.FilesystemIngest` (fields: `URI`, `Content`, `Cursor`, `Include`, `Exclude`, `Metadata`).
Passes through to the filesystem source connector via `shared.RunSourceIngest`.

## Upload

Accepts `multipart/form-data` with:

- `files` — one or more file parts.
- `paths` — optional relative paths (for browser folder uploads; one per file).

Files are staged under `FILESYSTEM_UPLOAD_ROOT/<upload-id>/` (default `storage/raw/uploads/<id>/`).
The following metadata keys are added to every result event:

| Key                               | Value                                                |
| --------------------------------- | ---------------------------------------------------- |
| `filesystem_upload_id`            | Random 32-hex-char ID                                |
| `filesystem_upload_root`          | Absolute path to the staged upload directory         |
| `filesystem_upload_file_count`    | Number of files uploaded                             |
| `filesystem_upload_original_name` | Relative path of the file (single-file uploads only) |

Path traversal and absolute paths are rejected with HTTP 400.
