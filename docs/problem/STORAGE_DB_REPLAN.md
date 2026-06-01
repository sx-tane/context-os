# Storage And Database Replan

This document explains how `storage/` and PostgreSQL are connected today, what is actually useful, what is misleading, and how to reset the architecture into something production-ready.

## Current Reality

ContextOS currently has two persistence systems:

1. PostgreSQL tables managed by `migrations/`.
2. Local files under `storage/`.

They are not equal. The UI mostly reads from PostgreSQL. The `storage/` folders are mostly side effects, caches, staging files, and debug snapshots.

## What The UI Reads Today

| UI / API Feature | Real Read Source |
| --- | --- |
| Workspace list | PostgreSQL `workspaces` |
| Workspace status counts | PostgreSQL `ingest_events`, `entities`, `mismatches`, `connector_syncs` |
| Chat local artifact answers | PostgreSQL `ingest_events` |
| Artifact search | PostgreSQL `ingest_events` |
| Graph panel | PostgreSQL `entities` |
| Findings results | Fresh pipeline result or PostgreSQL `mismatches` cache |
| Uploaded file browser staging | Local `storage/raw/uploads/` |
| Parsed document replay/debug | Local `storage/parsed/` |
| Embedding cache | Local `storage/embeddings/` |
| Graph snapshots | Local `storage/snapshots/` |

So if `storage/` has files but PostgreSQL is empty, chat/graph/artifacts can still show nothing.

## What Writes To Storage Today

| Path | Writer | Read By Product UI? | Current Role |
| --- | --- | --- | --- |
| `storage/raw/uploads/<upload-id>/` | `apps/api/handler/filesystem/filesystem_upload.go` | No, not directly | Browser upload staging before filesystem ingest |
| `storage/parsed/<workspace-id>/<event-id>.json` | `internal/normalization.DocumentWriter`, wired in `apps/api/main.go` | No | Debug/replay side output |
| `storage/embeddings/<sha256>.json` | `internal/aiworker.EmbeddingCache`, wired in `apps/api/main.go` | No | Local embedding cache |
| `storage/snapshots/<workspace-id>_<trace-id>.json` | `internal/graph.ContextGraph.SaveSnapshot`, called by `internal/pipeline.persistResult` | No | Debug/regression graph snapshot |
| PostgreSQL tables | `internal/store` repositories | Yes | Actual product query source |

## Current Local State Observed

Checked against the local Postgres container:

| Table | Rows |
| --- | ---: |
| `workspaces` | 4 |
| `ingest_events` | 5 |
| `entities` | 209 |
| `relationships` | 18,751 |
| `mismatches` | 4 |
| `connector_syncs` | 5 |
| `audit_log` | 0 |

Storage folder size:

| Folder | Approx Size | Meaning |
| --- | ---: | --- |
| `storage/raw/` | 452 KB | Uploaded file staging from tests/manual runs |
| `storage/parsed/` | 64 KB | Parsed JSON side outputs |
| `storage/embeddings/` | 816 KB | Embedding cache files |
| `storage/snapshots/` | 37 MB | Graph snapshots; one workspace snapshot is very large |
| `storage/` total | 39 MB | Mostly generated local runtime data |

Important finding: `audit_log` is empty. That means the system does not yet record a reliable operational history of ingest, graph, or findings actions.

Important finding: 5 ingest events produced 18,751 relationships. That is a warning sign. Relationship generation is too noisy for production and can make graph output feel useless.

## What Is Actually Useful Right Now

Useful:

- PostgreSQL `workspaces`, `ingest_events`, `entities`, `mismatches`, and `connector_syncs`.
- `/workspace/status`, `/artifacts`, `/chat/query`, `/graph`, and `/presentation/findings` when they are backed by Postgres.
- `storage/raw/uploads/` for browser upload staging.
- `storage/parsed/` for debugging how normalization output looks.

Weak or misleading:

- `storage/snapshots/` is not used by the UI and can become huge.
- `storage/embeddings/` is a cache, not product memory.
- `storage/raw/` does not currently represent every external source. It only reliably stages browser uploads.
- `audit_log` exists but is not meaningfully written.
- Graph relationships are too dense and need constraints before graph visualization becomes useful.

## Correct Mental Model

Use this rule until the architecture is cleaned:

```text
Postgres = source of truth for the product.
storage/raw = upload staging and future raw replay store.
storage/parsed = derived debug/replay output.
storage/embeddings = disposable cache.
storage/snapshots = disposable/debug graph snapshots until promoted.
```

The user-facing product should not claim something exists unless Postgres can prove it.

## Why The Structure Feels Like It Is Dying

The project currently mixes three ideas:

- product memory
- local cache
- debug artifacts

They all live under `storage/`, but they do not have the same lifecycle. Some are durable source material, some are derived, some are temporary. Because the UI reads Postgres while many files land in `storage/`, the developer experience becomes confusing.

The current folder names are not wrong, but the contracts are missing.

## Replanned Architecture

### Source Of Truth

PostgreSQL should be the only source of truth for the product UI:

- workspace identity
- source artifacts
- connector sync state
- entities
- relationships
- findings
- audit trail

### Local Files

Local files should be split by lifecycle:

```text
storage/
  runtime/
    uploads/        temporary upload staging
    cache/
      embeddings/   disposable embedding cache
  artifacts/
    raw/            durable raw source payloads, content-addressed
    parsed/         reproducible normalized documents
    snapshots/      bounded graph snapshots for replay/debug
  exports/          user-visible exports, reports, bundles
```

Do not implement this move blindly yet. First add tests and migration steps.

## Phase 0: Stop The Confusion

Goal: make current behavior visible and honest.

Tasks:

- Add a system state endpoint that returns DB counts and storage folder sizes.
- Show DB-backed status in the frontend: workspace events, entities, relationships, findings, sync rows.
- Label raw `/ingest` and connector debug pages as non-persistent until rewired.
- Add a storage cleanup command for generated ignored files.

Exit criteria:

- User can see whether data is in DB or only in local files.
- UI cannot say "ready" unless DB status confirms events for that source.

## Phase 1: Make Postgres The Product Truth

Goal: every product action writes the same DB-backed pipeline.

Tasks:

- Route all source setup and connector debug ingest through one persistent ingest service.
- Make `/ingest` accept workspace scope or explicitly rename it as `/ingest/preview`.
- Write real connector `event_count`.
- Write `audit_log` rows for ingest started, ingest completed, ingest failed, graph updated, findings detected.

Exit criteria:

- `/workspace/status`, `/artifacts`, `/chat/query`, and `/graph` agree after every ingest.
- `audit_log` explains what happened.

## Phase 2: Control Relationship Explosion

Goal: graph data should be useful instead of huge.

Tasks:

- Add relationship caps or stricter relationship rules.
- Stop building full adjacent co-occurrence relationship meshes for noisy documents.
- Store relationship provenance and confidence.
- Add graph density metrics per workspace.

Exit criteria:

- A small ingest cannot create thousands of low-value relationships.
- Graph panel shows explainable, inspectable relationships.

## Phase 3: Give Storage Real Contracts

Goal: make every `storage/` folder have a lifecycle.

Tasks:

- Define which files are durable and which are disposable.
- Store raw source payloads content-addressed when replay is required.
- Keep parsed and snapshots derived from DB/raw data.
- Add retention limits for snapshots and embeddings.
- Add cleanup commands:
  - clear runtime uploads
  - clear embedding cache
  - clear snapshots
  - keep raw durable artifacts

Exit criteria:

- Deleting cache/snapshot folders does not break product state.
- Durable raw artifacts can replay a workspace.

## Phase 4: Rebuild Replay

Goal: rebuild workspace state from durable source records.

Tasks:

- Add a replay command: workspace ID in, DB/raw artifacts out to graph/findings.
- Version normalized documents.
- Version graph snapshots.
- Add fixture-based replay tests.

Exit criteria:

- A workspace can be rebuilt deterministically.
- Replay results can be compared in tests.

## Phase 5: Production Storage Policy

Goal: make storage safe for long-term local use.

Tasks:

- Add backup/restore docs for Postgres.
- Add cleanup/retention docs for local files.
- Make secrets/token leakage checks part of ingestion.
- Add a "storage health" UI panel.

Exit criteria:

- User knows what data exists, where it lives, how big it is, and how to clear it safely.

## Immediate Recommendation

Do not build more product UI on top of the current graph until storage and relationship contracts are tightened.

Next practical implementation order:

1. Add system state endpoint.
2. Add frontend system state panel.
3. Rewire all ingest paths to one persistent service.
4. Add audit log writes.
5. Reduce relationship explosion.
6. Split `storage/` into runtime/cache/artifacts/exports contracts.
