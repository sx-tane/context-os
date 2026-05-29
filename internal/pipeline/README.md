# Internal Pipeline Orchestration

This folder contains orchestration code that wires stage implementations into executable pipeline flow.

## Responsibilities

- Assemble stage execution in the expected order.
- Carry trace identifiers and stage outputs end-to-end.
- Keep orchestration deterministic and observable.

## Maintenance Checklist

- Update this README when orchestration sequence or wiring changes.
- Ensure orchestration tests reflect stage ordering and error handling.
- Keep contract changes synced with `domain/pipelines` documentation.
