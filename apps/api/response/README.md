# API Response Types

This folder defines HTTP response payload contracts returned by API handlers.

## Responsibilities

- Keep response structs stable and well-documented.
- Preserve compatibility for API consumers when adding fields.
- Use dedicated error response shapes for non-2xx responses.

## Current Files

- `error.go`: shared error response envelope.
- `chat.go`: chat query response payload, including `provider` (`local` or `codex`) so clients can tell whether an answer came from persisted artifacts or live Codex source context.
- `ingest.go`: ingest response payload contract.

## Maintenance Checklist

- Update response types alongside route behavior updates.
- Reflect changed fields in API docs and handler tests.
- Avoid leaking internal stage-only fields to external clients.
