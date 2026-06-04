# API Response Types

This folder defines HTTP response payload contracts returned by API handlers.

## Responsibilities

- Keep response structs stable and well-documented.
- Preserve compatibility for API consumers when adding fields.
- Use dedicated error response shapes for non-2xx responses.

## Current Files

- `error.go`: shared error response envelope.
- `chat.go`: chat query response payload, including backward-compatible plain `answer`, structured `answer_sections` source cards, `provider` (`local` or `codex`) so clients can tell whether an answer came from persisted artifacts or live Codex source context, plus `evidence_save_status`, `evidence_event_count`, `evidence_save_error`, and evidence graph status/count fields so clients can show whether concrete live answers saved into the Local DB and updated Graph.
- `ingest.go`: ingest response payload contract.
- `presentation.go`: presentation findings response payload, including event, entity, relationship, mismatch, severity, role-view, PMO, and execution evidence counts for frontend findings summaries.

## Maintenance Checklist

- Update response types alongside route behavior updates.
- Reflect changed fields in API docs and handler tests.
- Avoid leaking internal stage-only fields to external clients.
