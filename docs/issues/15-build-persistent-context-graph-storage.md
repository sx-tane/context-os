# Issue 15: Build persistent context graph storage

## Description

Store entities, aliases, relationships, history, and snapshots using PostgreSQL, pgvector, and filesystem storage.

## Acceptance criteria

- Context graph model stores canonical entities and relationships.
- Storage folders exist for raw, parsed, embeddings, and snapshots.
- Database migrations can be added incrementally in `migrations/`.
