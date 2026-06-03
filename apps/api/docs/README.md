# API Docs Bundle

This folder stores generated Swagger artifacts served or inspected with the API app.

## Generated Artifacts

- `docs.go`: generated docs package registration.
- `swagger.json`: machine-readable OpenAPI spec.
- `swagger.yaml`: YAML OpenAPI spec.
- `api.html`: rendered documentation page.

## Update Workflow

- Regenerate after handler route or request/response schema changes.
- Treat this folder as the checked-in OpenAPI source for frontend codegen.
- Validate generated docs compile in API tests.

## Recent Regeneration

Last regenerated to include workspace-scoped source, chat, graph, and presentation contracts:

- Ingest request schemas include `workspace_id` so direct and Codex-backed sources persist into the active workspace.
- `request.ChatQuery`, `repository.Workspace`, and `repository.ConnectorSync` are present for local chat, workspace status, and connector sync responses.
- `response.ChatQuery.provider` indicates whether chat answered from local artifacts or Codex-backed live source context.
- `response.ChatQuery.evidence_save_status`, `evidence_event_count`, and `evidence_save_error` document live-chat evidence persistence into the Local DB.
- `/graph` documents flattened graph entities and persisted relationship edges for frontend graph rendering.
- Google Drive connector docs include status and ingest endpoints for Docs, Sheets, and Slides sources.

## Verification

- API route tests should pass after regeneration.
- Swagger output should include newly added routes and tags.
