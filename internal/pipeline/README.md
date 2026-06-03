# Internal Pipeline Orchestration

This folder contains orchestration code that wires stage implementations into executable pipeline flow.

## Responsibilities

- Assemble stage execution in the expected order.
- Carry trace identifiers and stage outputs end-to-end.
- Keep orchestration deterministic and observable.
- Persist events, entities, relationships, mismatches, and filesystem snapshots when repository stores are provided.
- Support a graph-only run for already-returned live chat evidence so Activity and Graph can update without running Findings.

## Persistence

Pipeline persistence uses a detached 30 second timeout for each repository write. This keeps persistence bounded while allowing final writes to complete even if the request context has already been cancelled by the caller.

Event persistence reports the number of written event rows, including both new
rows and duplicate rows updated during idempotent replays.

`RunEvents` executes the full post-ingest flow through reasoning. `RunEventsGraphOnly` executes normalization, classification, extraction, identity, relationship, and graph persistence for events that were already returned by a live connector answer, but skips reasoning/mismatch persistence. This keeps live chat evidence saves from auto-running Findings while still updating the persisted graph.

## Maintenance Checklist

- Update this README when orchestration sequence or wiring changes.
- Ensure orchestration tests reflect stage ordering and error handling.
- Keep contract changes synced with `domain/pipelines` documentation.
- Keep persistence timeout changes covered by pipeline tests when cancellation or repository write behavior changes.
