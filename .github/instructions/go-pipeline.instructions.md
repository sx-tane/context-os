---
description: "Use when implementing or refactoring Go code in domain and internal pipeline modules. Enforces stage boundaries, contracts-first design, and explainable outputs."
applyTo: "{domain,internal,tests}/**/*.go"
---
# Go Pipeline Instruction

- Keep domain models and contracts in domain packages.
- Keep concrete behavior and orchestration in internal packages.
- Avoid leaking source-specific payload shapes beyond ingestion and normalization.
- Ensure every stage output can be traced to stage input identifiers.
- Keep functions small and explicit about confidence and provenance fields when present.
- Add tests for cross-stage behavior changes, not only unit happy paths.
