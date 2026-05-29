# Parsed Storage

Stores normalized and parsed documents produced after ingestion and extraction.

This folder currently contains only this README. Add parsed artifacts here when stage output snapshots are materialized for debugging or regression review.

## Responsibilities

- Hold deterministic intermediate outputs suitable for downstream stages.
- Preserve enough structure to inspect parsing behavior during debugging.
- Keep parsed artifacts distinct from raw uploads and derived embeddings.

## Maintenance Checklist

- Document new parsed artifact formats when introduced.
- Prefer deterministic filenames for regression and benchmark comparisons.
- Keep parsing expectations aligned with extraction and normalization docs.
