---
name: contextos-pipeline-stage-delivery
description: "Implement or refactor a ContextOS pipeline stage with contracts, events, traceability, and tests. Use when: changing ingestion, normalization, classification, extraction, identity, relationship, graph, reasoning, execution, or presentation behavior; updating stage contracts; adding traceable stage outputs. Covers contracts, events, provenance, tests, and downstream compatibility."
argument-hint: "Which stage and what behavior should be delivered?"
user-invocable: true
---

# ContextOS Pipeline Stage Delivery

## Outcome

Deliver one pipeline-stage change that is traceable, testable, and aligned with ContextOS domain boundaries.

## When to Use

- Building a new stage capability.
- Refactoring stage behavior without breaking contracts.
- Adding stage outputs required by downstream reasoning.

## Procedure

1. Scope the stage and define the target input and output behavior.
2. Confirm contract impact in domain packages.
3. Implement internal stage logic with explicit trace identifiers.
4. Emit or update events for stage transitions.
5. Add tests for normal flow and failure or ambiguity flow.
6. Update the nearest stage or contract `README.md` when behavior, contracts, events, or run commands change.
7. Validate downstream compatibility.

## Decision Points

- If contract changes are required: update domain first, then internal implementations.
- If ambiguity exists in extracted meaning: preserve confidence and provenance instead of forcing certainty.
- If stage output is source-specific: normalize before handing off.

## Completion Checks

- Stage behavior can be explained with deterministic steps.
- Output includes identifiers needed for traceability.
- Tests cover at least one error or conflict path.
- Downstream stage assumptions remain valid.
- Relevant stage, contract, or package README is aligned with the implementation.

## References

- [Stage Checklist](./references/stage-checklist.md)
- [Stage Test Template](./assets/stage-test-template.md)
