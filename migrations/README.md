# Migrations

Database migration files for PostgreSQL and pgvector-backed local persistence.

This folder stores ordered migration files for local persistence schemas.

Current migrations:

- `0001_enable_pgvector.sql` — enables the `vector` extension for local Postgres instances.

## Responsibilities

- Track schema changes in an ordered, reproducible form.
- Keep local persistence aligned with application expectations.
- Document new tables, indexes, or extension requirements when introduced.

## Maintenance Checklist

- Add migration notes when schema shape changes materially.
- Keep migration filenames ordered and deterministic.
- Update local setup or production readiness docs if migration prerequisites change.
