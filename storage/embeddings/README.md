# Embeddings Storage

Stores local embedding artifacts and pgvector-related snapshots.

This folder currently contains only this README. Add embedding artifacts here when local vector generation or pgvector export workflows are introduced.

## Responsibilities

- Hold reproducible embedding outputs generated during local workflows.
- Separate derived vector artifacts from raw and parsed source material.
- Keep embeddings traceable to their source documents or snapshots.

## Maintenance Checklist

- Document new embedding file formats here.
- Avoid storing secrets or unstable transient files in tracked artifacts.
- Keep retention or regeneration guidance aligned with storage policy.
