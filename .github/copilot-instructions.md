# ContextOS Project Guidelines

## Product Direction
- Optimize for understanding synchronization across business and engineering.
- Prioritize local-first personal workflow over SaaS or multi-tenant concerns.
- Keep the first success metric in focus: detect real FE and BE misunderstanding automatically.

## Architecture Boundaries
- Respect layered flow: source -> ingestion -> normalization -> classification -> extraction -> identity -> relationship -> graph -> reasoning -> execution -> presentation.
- Keep domain contracts stable in domain and put implementations in internal.
- Preserve event-driven behavior between stages with traceable identifiers.

## Implementation Rules
- Prefer deterministic and explainable behavior where possible.
- Treat AI output as assistive evidence, never as blind source of truth.
- Preserve provenance links from findings back to source artifacts.
- Do not introduce broad abstractions before the pipeline need is clear.

## Quality Bar
- Add or update tests for pipeline behavior changes.
- Surface confidence, impact, and evidence when implementing misalignment logic.
- For connector work, include idempotency and replay safety checks.
