# ContextOS Project Guidelines

## Product Direction

- Optimize for understanding synchronization across business and engineering.
- Prioritize local-first personal workflow over SaaS or multi-tenant concerns.
- Keep the first success metric in focus: detect real cross-layer context misalignment automatically.

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

## Explanation Style

- When explaining architecture, workflows, pipeline stages, skill routing, state transitions, or multi-step behavior, include a Mermaid diagram so the relationship is visually clear.
- Keep the diagram small and purposeful; skip only for trivial one-line answers, pure command output, or cases where a diagram would be misleading.

## Clarifying Ambiguous Requests

- If the user request is ambiguous, restate the interpreted prompt in one short sentence and ask the minimum clarifying question before editing files.
- If the intent is clear enough to act safely, proceed without asking and keep the work scoped to the interpreted request.
