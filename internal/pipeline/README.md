# Internal Pipeline Orchestration

This folder contains orchestration code that wires stage implementations into executable pipeline flow.

## Responsibilities

- Assemble stage execution in the expected order.
- Carry trace identifiers and stage outputs end-to-end.
- Keep orchestration deterministic and observable.
- Persist events, entities, relationships, mismatches, and filesystem snapshots when repository stores are provided.

## Persistence

Pipeline persistence uses a detached 30 second timeout for each repository write. This keeps persistence bounded while allowing final writes to complete even if the request context has already been cancelled by the caller.

## Maintenance Checklist

- Update this README when orchestration sequence or wiring changes.
- Ensure orchestration tests reflect stage ordering and error handling.
- Keep contract changes synced with `domain/pipelines` documentation.
- Keep persistence timeout changes covered by pipeline tests when cancellation or repository write behavior changes.
