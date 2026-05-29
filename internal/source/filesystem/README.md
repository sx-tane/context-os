# Filesystem Source

Local-first source connector for files and folders already present on disk or supplied as inline content.

## What It Does

- Exposes the public connector name `filesystem` with `files` capability.
- Emits one `document.ingested` event per file, including one event per supported file found during recursive folder ingestion.
- Uses local path, content hash, file size, modified time, and extension metadata for replay.
- Applies optional include/exclude path rules before reading local files or folder children.
- Skips unsupported, oversized, unreadable, and symlinked folder children while recording skip counts and the first skipped path in metadata.

## File Layout

| File                 | Role                                                                             |
| -------------------- | -------------------------------------------------------------------------------- |
| `filesystem.go`      | Parent connector, request validation, dispatch, path rules, metadata enrichment. |
| `text.go`            | Text/code/config detection and direct text extraction.                           |
| `spreadsheet.go`     | `.csv` and `.xlsx` workbook, sheet, row, cell, and formula extraction.           |
| `openapi.go`         | OpenAPI/Swagger JSON/YAML detection and endpoint/schema/enum metadata.           |
| `office.go`          | `.docx` and `.pptx` XML text extraction.                                         |
| `pdf.go`             | Best-effort PDF literal and stream text extraction.                              |
| `archive.go`         | Shared zip archive helper used by Office and spreadsheet formats.                |
| `filesystem_test.go` | Behavior tests for all supported file types and provenance.                      |

## Supported Formats

| Format            | Extensions                                      | Metadata Notes                                                   |
| ----------------- | ----------------------------------------------- | ---------------------------------------------------------------- |
| Folder            | Directory path                                  | Recurses deterministically and emits one event per child file.   |
| Text and Markdown | `.txt`, `.md`                                   | `filesystem_format=text`                                         |
| Code and config   | `.go`, `.ts`, `.json`, `.yaml`, `.toml`, `.sql` | Read directly; JSON/YAML also checked for OpenAPI markers.       |
| Spreadsheet       | `.csv`, `.xlsx`                                 | Adds `filesystem_spreadsheet_*` keys.                            |
| OpenAPI spec      | `.json`, `.yaml`, `.yml`                        | Adds `openapi_*` summary keys while remaining a filesystem file. |
| Word              | `.docx`                                         | Extracts paragraph/header/footer text.                           |
| PDF               | `.pdf`                                          | Best-effort literal text extraction.                             |
| PowerPoint        | `.pptx`                                         | Extracts slide text.                                             |

## Folder Metadata And Limits

Folder ingestion keeps each child event identified as a file while adding folder context:

- `filesystem_ingest_mode=folder`
- `filesystem_root`
- `filesystem_relative_path`
- `filesystem_folder_file_count`
- `filesystem_folder_skipped_count`
- `filesystem_folder_first_error`

Optional guardrail metadata:

- `filesystem_max_files` defaults to `1000`.
- `filesystem_max_file_size` defaults to `10485760` bytes.

If no supported files are found, the connector returns a structured invalid-request error with the first skipped path when available.

## Boundary

Spreadsheet and OpenAPI are file types, not standalone source packages. Keep new file-format logic in this package unless it becomes a separate external system.
