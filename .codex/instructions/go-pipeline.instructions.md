---
description: "Use when implementing or refactoring Go code in domain and internal pipeline modules. Enforces stage boundaries, contracts-first design, and explainable outputs."
applyTo: "{domain,internal,tests}/**/*.go"
---

# Go Pipeline Instruction

## Skills

- For Go code quality, apply the **go-best-practices** skill.
- For writing or reviewing `_test.go` files, apply the **go-test-patterns** skill.

## Rules

- Keep domain models and contracts in domain packages.
- Keep concrete behavior and orchestration in internal packages.
- Avoid leaking source-specific payload shapes beyond ingestion and normalization.
- Ensure every stage output can be traced to stage input identifiers.
- Keep functions small and explicit about confidence and provenance fields when present.
- Add tests for cross-stage behavior changes, not only unit happy paths.
- Update the nearest stage, contract, or package README when behavior, contracts, events, or run commands change.
