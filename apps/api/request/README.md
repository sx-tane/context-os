# API Request Types

This folder defines HTTP request payload contracts consumed by API handlers.

## Responsibilities

- Keep request structs explicit and small.
- Align JSON tags with handler decoding expectations.
- Avoid embedding stage-internal implementation details.

## Current Files

- `ingest.go`: request shape used by ingest endpoints.

## Maintenance Checklist

- Update request structs when endpoint inputs change.
- Regenerate swagger docs if request contracts are modified.
- Add or update handler tests when validation behavior changes.
