# Snapshot Storage

Stores historical context graph snapshots and comparable baseline outputs.

The context graph writes deterministic JSON snapshots here via
`graph.(*ContextGraph).SaveSnapshot(dir, name)` and reloads them with
`graph.LoadSnapshot(path)`. Each snapshot captures the current entities and
relationships plus their full version history so a run can be replayed or audited.

## Snapshot Format

- File name: `<name>.json` (caller-supplied name, e.g. `run-2026-06-01.json`).
- Top-level `schema_version` field guards replay; unknown versions are rejected on load.
- Output is indented and key-ordered by `encoding/json`, so the same graph state always
  produces byte-identical files suitable as regression baselines.

## Responsibilities

- Capture stable point-in-time outputs for comparison and regression review.
- Support local analysis of graph evolution and reasoning changes.
- Keep snapshot naming and contents deterministic where possible.

## Maintenance Checklist

- Document snapshot naming or retention rules when they change.
- Keep snapshots free of volatile values when used for regression comparison.
- Align snapshot usage with harness and storage documentation.
- Bump `snapshotSchemaVersion` in `internal/graph/snapshot.go` when the on-disk shape changes.
