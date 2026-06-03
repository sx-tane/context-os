# ContextOS Codex Instructions

Codex should use this file as the repo-level operating guide. The previous GitHub Copilot customization remains in `.github/`; Codex-facing mirrors live in `.codex/`.

## Product Direction

- Optimize for understanding synchronization across business and engineering.
- Prioritize local-first personal workflow over SaaS or multi-tenant concerns.
- Keep the first success metric in focus: detect real cross-layer context misalignment automatically.

## Architecture Boundaries

- Respect layered flow: source -> ingestion -> normalization -> classification -> extraction -> identity -> relationship -> graph -> reasoning -> execution -> presentation.
- Keep domain contracts stable in `domain/` and put implementations in `internal/`.
- Preserve event-driven behavior between stages with traceable identifiers.
- Keep internal stage packages independent; use `domain/` contracts as the bridge.

## Default Agent Mode

Use the implementer behavior for coding, tests, connectors, handlers, harnesses, docs, and customization edits. The migrated source is `.codex/agents/contextos-implementer.agent.md`.

Use the architect behavior only when the user asks for planning, sequencing, dependency mapping, tradeoffs, or issue breakdown without edits. The migrated source is `.codex/agents/contextos-architect.agent.md`.

## Skill Routing

Load the relevant skill from `.codex/skills/<skill>/SKILL.md` before specialized work:

- `go-best-practices`: Go implementation, review, and refactor quality.
- `go-test-patterns`: Go `_test.go` files and test style review.
- `contextos-pipeline-stage-delivery`: pipeline stage behavior, contracts, events, and traceability.
- `contextos-api-handler`: API handlers, source connector routes, status, ingest, and stream endpoints.
- `contextos-frontend-design`: Svelte page/component UI design, spacing, buttons, panels, graph views, source setup, and chat visual consistency.
- `contextos-frontend-connector`: Svelte connector components and connector registration.
- `frontend-jest-swc-patterns`: frontend `*.test.ts` files, `$lib` mocks, fetch mocks, and setter lifecycle tests.
- `contextos-harness-engineering`: fixtures, scenarios, goldens, benchmarks, and regression gates.
- `contextos-identity-resolution-benchmark`: identity merge rules, thresholds, aliases, and matching metrics.
- `contextos-misalignment-report`: cross-layer mismatch reports with evidence, confidence, impact, and recommended action.
- `contextos-issue-workflow`: GitHub parent-child issue groups and labels.
- `contextos-authoring`: new or changed skills, instruction files, agents, or routing maps.

If a mirrored skill is unavailable, use the fallback rules in `.codex/agents/contextos-implementer.agent.md` or `.codex/agents/contextos-architect.agent.md`.

## File-Scoped Guidance

The Copilot `applyTo` globs were migrated into `.codex/instructions/`. Codex should apply the same intent when touching matching files:

- `apps/api/**/*.go`: follow API handler instructions and use `contextos-api-handler`.
- `internal/source/**/*.go`: follow connector instructions and use `contextos-api-handler`.
- `{domain,internal,tests}/**/*.go`: follow Go pipeline instructions and use `go-best-practices` plus `go-test-patterns`.
- `internal/{reasoning,presentation,graph}/**/*.go`: follow reasoning output instructions and use `contextos-misalignment-report`.
- `apps/frontend/src/**/*.test.ts`: follow frontend test instructions and use `frontend-jest-swc-patterns`.
- `apps/frontend/src/**/*.svelte`: follow frontend design instructions and use `contextos-frontend-design`; add `contextos-frontend-connector` only for connector components.
- `.github/{agents,instructions,skills}/**/*.md` and `.codex/{agents,instructions,skills}/**/*.md`: follow customization authoring instructions and use `contextos-authoring`.

## Implementation Rules

- Prefer deterministic and explainable behavior where possible.
- Treat AI output as assistive evidence, never as blind source of truth.
- Preserve provenance links from findings back to source artifacts.
- Do not introduce broad abstractions before the pipeline need is clear.
- Keep changes scoped to requested behavior.
- Add or update tests for behavior changes.
- Update the nearest relevant `README.md` whenever code, contracts, behavior, setup, workflows, or customization routing changes.

## Go Quality Rules

- Handle errors first with guard clauses; avoid `if err == nil` nesting.
- Accept narrow interfaces, not concrete types.
- Public stage APIs should be synchronous; callers decide concurrency.
- Every goroutine spawned in a stage must respect `ctx.Done()` or a quit channel.
- Exported identifiers need doc comments starting with the identifier name.
- Every reasoning output struct touched in this repo must include populated `Evidence []string` and `Confidence float64` fields when applicable.

## Verification

- For Go changes, run `go test ./...` and `go vet ./...` unless the touched scope justifies a narrower package command.
- For frontend changes, run `bun run test` and `bun run check` from `apps/frontend/` when relevant.
- For skill or customization routing changes, run:

```bash
.codex/skills/contextos-authoring/scripts/score-skills.sh
.codex/skills/contextos-authoring/scripts/score-skill-routing.sh
.codex/skills/contextos-authoring/scripts/check-mermaid-policy.sh
.codex/skills/contextos-authoring/scripts/score-readme-coverage.sh
.codex/skills/contextos-authoring/scripts/score-readme-quality.sh
.codex/skills/contextos-authoring/scripts/check-readme-sync-on-change.sh
```

## Explanation Style

When explaining architecture, workflows, pipeline stages, skill routing, state transitions, or multi-step behavior, include a small Mermaid diagram unless the answer is trivial, pure command output, or a diagram would be misleading.

## Ambiguity

If a request is ambiguous, restate the interpreted prompt in one short sentence and ask the minimum clarifying question before editing files. If the intent is clear enough to act safely, proceed without asking and keep the work scoped to the interpreted request.
