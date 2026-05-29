# Migrations

Database migration files for PostgreSQL and pgvector-backed local persistence.

## Responsibilities

- Track schema changes in an ordered, reproducible form.
- Keep local persistence aligned with application expectations.
- Document new tables, indexes, or extension requirements when introduced.

## Maintenance Checklist

- Add migration notes when schema shape changes materially.
- Keep migration filenames ordered and deterministic.
- Update local setup or production readiness docs if migration prerequisites change.
