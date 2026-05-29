# API Response Types

This folder defines HTTP response payload contracts returned by API handlers.

## Responsibilities

- Keep response structs stable and well-documented.
- Preserve compatibility for API consumers when adding fields.
- Use dedicated error response shapes for non-2xx responses.

## Current Files

- `error.go`: shared error response envelope.
- `ingest.go`: ingest response payload contract.

## Maintenance Checklist

- Update response types alongside route behavior updates.
- Reflect changed fields in API docs and handler tests.
- Avoid leaking internal stage-only fields to external clients.
