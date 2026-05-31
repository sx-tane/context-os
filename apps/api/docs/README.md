# API Docs Bundle

This folder stores generated Swagger artifacts served or inspected with the API app.

## Generated Artifacts

- `docs.go`: generated docs package registration.
- `swagger.json`: machine-readable OpenAPI spec.
- `swagger.yaml`: YAML OpenAPI spec.
- `api.html`: rendered documentation page.

## Update Workflow

- Regenerate after handler route or request/response schema changes.
- Keep this folder in sync with `apps/api/_docs/` when generation output changes.
- Validate generated docs compile in API tests.

## Recent Regeneration

Last regenerated to include the Google Drive connector endpoints:
- `GET /googledrive/status` — reports OAuth/service-account/folder configuration
- `POST /googledrive/ingest` — ingests Docs, Sheets, and Slides from a Drive folder
- `request.GoogleDriveIngest` schema with `uri`, `folder_id`, `credential_path`, `service_account_path`, `access_token`, and `cursor`

## Verification

- API route tests should pass after regeneration.
- Swagger output should include newly added routes and tags.
