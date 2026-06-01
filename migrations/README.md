# Migrations

Database migration files for PostgreSQL and pgvector-backed local persistence.

This folder stores ordered migration files for local persistence schemas.
Migrations are embedded into the API binary at compile time via `migrations/migrations.go`
and applied automatically on startup by `storage/db.Open()`.

## Current migrations

| File | Purpose |
|------|---------|
| `0001_enable_pgvector.sql` | Enables the `vector` extension for local Postgres instances. |
| `0002_workspace_schema.sql` | Core persistence layer: workspaces, ingest_events, entities, relationships, mismatches, connector_syncs, audit_log. |

## Schema overview (0002)

- **workspaces** — one row per local project root; `path` is the unique key.
- **ingest_events** — raw source events captured per workspace; `UNIQUE(id, workspace_id)` makes ingestion idempotent.
- **entities** — identity-resolved entities per workspace; confidence is updated on re-ingestion.
- **relationships** — typed edges between entities; confidence is merged upward on re-ingestion.
- **mismatches** — reasoning findings per workspace with evidence and trace_id.
- **connector_syncs** — replay cursor and last-sync timestamp per `(workspace_id, connector, source_uri)`.
- **audit_log** — immutable append-only event log per workspace.

## Responsibilities

- Track schema changes in an ordered, reproducible form.
- Keep local persistence aligned with application expectations.
- Document new tables, indexes, or extension requirements when introduced.

## Maintenance Checklist

- Add migration notes when schema shape changes materially.
- Keep migration filenames ordered and deterministic.
- Update local setup or production readiness docs if migration prerequisites change.
