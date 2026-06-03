---
name: contextos-authoring
description: "Create a new ContextOS skill, instruction file, or agent following the established repo pattern. Use when: adding a new SKILL.md; adding a .instructions.md file; adding or updating a .agent.md; deciding which customization primitive to use; wiring a new skill into an agent. Covers naming conventions, folder layout, frontmatter requirements, assets/references structure, agent tool-list shape, and wiring checklist."
argument-hint: "What do you want to create? (skill / instruction / agent)"
user-invocable: true
---

# ContextOS Authoring Skill

## Outcome

Deliver a correctly-placed, fully-wired customization primitive that is immediately discoverable by the agent:

| What you want       | Deliverable                                                                           |
| ------------------- | ------------------------------------------------------------------------------------- |
| New **skill**       | `.github/skills/<name>/SKILL.md` + `assets/` + `references/` + README alignment       |
| New **instruction** | `.github/instructions/<name>.instructions.md` with `applyTo` glob + README alignment  |
| New **agent**       | `.github/agents/<name>.agent.md` with `tools:` list + skill wiring + README alignment |

---

## Decision Table — Which Primitive?

| Need                                                  | Use                              |
| ----------------------------------------------------- | -------------------------------- |
| Always-on, applies to all work in the repo            | `copilot-instructions.md`        |
| Applied automatically when editing specific files     | **Instruction** (`applyTo` glob) |
| Loaded on demand for a specialized multi-step task    | **Skill**                        |
| Isolated agent mode with different tool restrictions  | **Agent**                        |
| One-shot parameterised task (e.g. "generate X for Y") | Prompt (`.prompt.md`)            |

---

## Documentation Alignment

Every customization change must keep the matching README current:

- Update `.github/README.md` when an agent, instruction, skill, wiring map, folder structure, or routing rule changes.
- Update the nearest product or package README when the customization changes how files in that folder should be edited.
- Keep skeleton, checklist, example, and script links valid after every rename or move.
- Add `scripts/` only when the skill has a real executable helper; do not create empty script placeholders.
- Preserve the repo-wide Mermaid explanation policy when changing prompts, agents, or instruction files.

---

## Naming Conventions

| Prefix                                        | For                                                              |
| --------------------------------------------- | ---------------------------------------------------------------- |
| `contextos-*`                                 | ContextOS domain skills (pipeline stages, connectors, workflows) |
| `go-*`                                        | Go language quality skills (test patterns, best practices)       |
| `go-pipeline` / `api-handlers` / `connectors` | Instruction file names (kebab-case, no prefix needed)            |
| `contextos-*`                                 | Agent file names                                                 |

---

## Procedure — Create a Skill

1. **Choose a name** — kebab-case, max 64 chars, must match the folder name exactly.

2. **Create the folder and SKILL.md** at `.github/skills/<name>/SKILL.md`.
   Use the [skill skeleton](./assets/skill-skeleton.md). Required frontmatter:

   ```yaml
   ---
   name: <name> # must match folder
   description: "..." # discovery surface — include trigger phrases with "Use when:"
   argument-hint: "..." # hint shown when invoked via slash command
   user-invocable: true
   ---
   ```

3. **Add assets** in `.github/skills/<name>/assets/`.
   Typically one or more `*-skeleton.md` files with copy-paste code templates.
   Every skeleton snippet must be annotated with `<!-- only if ... -->` delete hints.

4. **Add references** in `.github/skills/<name>/references/`.
   Typically one `*-checklist.md` file with numbered completion criteria.

5. **Add scripts only when useful** in `.github/skills/<name>/scripts/`.
   Create runnable helpers only when the skill benefits from them; otherwise omit `scripts/` entirely.

6. **Wire the skill into agents**:
   - Open `.github/agents/contextos-implementer.agent.md`.
   - Add a `## <Feature> Delivery` section that invokes the skill by name,
     links to the skeleton and checklist, and provides inline fallback rules.
   - If the skill is architectural, also add a guidance section to
     `.github/agents/contextos-architect.agent.md`.

7. **Update README maps**:
   - Update `.github/README.md` with the new skill, instruction, or agent routing.
   - Update the nearest target-folder README if the skill changes how work in that folder is done.

8. **Validate** with the [authoring checklist](./references/authoring-checklist.md).
   For a repo-wide score, run [score-skills.sh](./scripts/score-skills.sh).

---

## Procedure — Create an Instruction File

1. **Choose a name and glob** — use the scope table below.

   | Scope                               | `applyTo` glob                                    |
   | ----------------------------------- | ------------------------------------------------- |
   | All Go in API app                   | `apps/api/**/*.go`                                |
   | All Go in source connectors         | `internal/source/**/*.go`                         |
   | All Go in domain + internal + tests | `{domain,internal,tests}/**/*.go`                 |
   | Svelte frontend                     | `apps/frontend/**/*.svelte`                       |
   | Frontend TypeScript tests           | `apps/frontend/src/**/*.test.ts`                  |
   | Reasoning / graph / presentation    | `internal/{reasoning,presentation,graph}/**/*.go` |
   | Customization files                 | `.github/{agents,instructions,skills}/**/*.md`    |

2. **Create the file** at `.github/instructions/<name>.instructions.md`.
   Use the [instruction skeleton](./assets/instruction-skeleton.md). Required frontmatter:

   ```yaml
   ---
   description: "Use when ..."
   applyTo: "<glob>"
   ---
   ```

3. **Body structure** — use these sections in order:
   - `## Skill` — reference the relevant skill if one exists.
   - `## <Concept> Shape` — structural rules (package layout, struct shape, naming).
   - `## <Checklist>` — ordered steps for common operations.
   - `## Do Not` — explicit anti-patterns.

4. **Keep it short** — instructions load automatically on every file match. Aim for < 50 lines. Deep detail belongs in the skill, not the instruction.

5. **Update README maps** — update `.github/README.md` and the nearest affected folder README when the instruction changes editing behavior.

---

## Procedure — Create an Agent

1. **Choose a name and role** — kebab-case, e.g. `contextos-<role>.agent.md`.

2. **Create the file** at `.github/agents/<name>.agent.md`.
   Use the [agent skeleton](./assets/agent-skeleton.md). Required frontmatter:

   ```yaml
   ---
   description: "Use for ..."
   name: "<Display Name>"
   tools: <comma-separated tool list>
   user-invocable: true
   ---
   ```

3. **Wire existing skills** — for each skill the agent should use, add a `## <Feature> Delivery` section:

   ```markdown
   ## <Feature> Delivery

   When doing <task>, apply the **<skill-name>** skill.
   Use the [skeleton](../skills/<skill-name>/assets/<x>-skeleton.md) for new files.
   Run the [checklist](../skills/<skill-name>/references/<x>-checklist.md) before marking complete.
   If the skill is unavailable, enforce these inline rules:

   - <rule 1>
   - <rule 2>
   ```

4. **Include a `## Procedure`** section with numbered steps the agent must follow for its primary workflow.

5. **Include a `## Constraints`** section with explicit guardrails (what must never happen).

6. **Update README maps** — update `.github/README.md` with the new or changed agent wiring.

7. **Validate** with the [authoring checklist](./references/authoring-checklist.md).

---

## Existing Skills Reference

| Name                                      | Triggers                                                                |
| ----------------------------------------- | ----------------------------------------------------------------------- |
| `contextos-api-handler`                   | new handler, `/connector/status`, `/connector/ingest`, source connector |
| `contextos-authoring`                     | new skill, instruction file, agent, wiring map                          |
| `contextos-frontend-design`               | Svelte UI design, spacing, buttons, panels, graph/source/chat visuals   |
| `contextos-frontend-connector`            | new `*Connector.svelte`, connector card, AbortController, runIngest     |
| `contextos-harness-engineering`           | fixtures, scenarios, goldens, benchmarks, regression gates              |
| `contextos-identity-resolution-benchmark` | identity resolution, alias merge, entity matching                       |
| `contextos-issue-workflow`                | GitHub issue, parent-child, epic, type:task                             |
| `contextos-misalignment-report`           | misalignment finding, cross-layer, confidence, impact                   |
| `contextos-pipeline-stage-delivery`       | pipeline stage, contracts, events, ingestion, normalization             |
| `frontend-jest-swc-patterns`              | `*.test.ts`, `$lib` mocks, fetch mocks, setter lifecycle tests          |
| `go-best-practices`                       | Go code quality, error handling, interface design, goroutines           |
| `go-test-patterns`                        | `_test.go`, doc comments, table-driven tests, `t.Fatalf`, subtests      |

---

## Skill Quality Benchmark

Run the authoring benchmarks after adding or changing skills:

```bash
.github/skills/contextos-authoring/scripts/score-skills.sh
.github/skills/contextos-authoring/scripts/score-skill-routing.sh
.github/skills/contextos-authoring/scripts/check-mermaid-policy.sh
.github/skills/contextos-authoring/scripts/score-readme-coverage.sh
.github/skills/contextos-authoring/scripts/score-readme-quality.sh
.github/skills/contextos-authoring/scripts/check-readme-sync-on-change.sh
```

The structural benchmark scores each skill out of 100 using structural, routing, documentation, and overlap checks:

- Frontmatter: folder/name match, `Use when` discovery text, argument hint, and `user-invocable`.
- Body: `Outcome`, `Procedure`, decision context, and `References` sections.
- Support files: non-empty `assets/` and `references/`; `scripts/` only when useful and referenced.
- Routing/docs: listed in `.github/README.md`, wired to at least one agent, and includes README alignment.
- Overlap: flags exact duplicate descriptions so redundant skills are visible.

The routing benchmark checks real prompt scenarios against expected skills. The Mermaid policy benchmark verifies explanatory response rules stay wired into repo-wide instructions and docs.

The README coverage benchmark scores tracked directory coverage and lists missing `README.md` folders.
The README quality benchmark scores folder documentation against the actual code context in that directory: direct child files, sibling folders, high-level architecture paths, and operational entrypoints.
The change-sync benchmark checks that code edits are accompanied by meaningful nearest-README updates in the selected git diff range. New code files must be reflected by the nearest README, and shallow README edits are rejected.

Passing bar: every structural skill score is at least 90, every routing scenario scores 100, the Mermaid policy score is 100, README coverage reaches 100, every required README quality score is 100, and the change-sync benchmark passes for the working diff.

## References

- [Skill Skeleton](./assets/skill-skeleton.md) — starting point for new skills.
- [Instruction Skeleton](./assets/instruction-skeleton.md) — starting point for new instruction files.
- [Agent Skeleton](./assets/agent-skeleton.md) — starting point for new agents.
- [Authoring Checklist](./references/authoring-checklist.md) — review before marking customization work complete.
- [Skill Score Script](./scripts/score-skills.sh) — repo-wide skill benchmark.
- [Skill Routing Scenarios](./references/skill-routing-scenarios.tsv) — prompt-to-skill benchmark cases.
- [Skill Routing Script](./scripts/score-skill-routing.sh) — validates routing scenario coverage.
- [Mermaid Policy Script](./scripts/check-mermaid-policy.sh) — validates visual explanation policy wiring.
- [README Coverage Exclusions](./references/readme-coverage-exclusions.txt) — minimal allowlist for non-doc directories.
- [README Coverage Script](./scripts/score-readme-coverage.sh) — validates folder README coverage.
- [README Quality Script](./scripts/score-readme-quality.sh) — validates README content quality against directory-specific code context.
- [README Change Sync Script](./scripts/check-readme-sync-on-change.sh) — validates README updates are included with code changes.

---

## Wiring Map — Agents × Skills

When a new skill is created, wire it to the correct agent section:

| Skill type                                               | Agent to update                                                             |
| -------------------------------------------------------- | --------------------------------------------------------------------------- |
| Implementation: handler, connector, frontend test, stage | `contextos-implementer.agent.md`                                            |
| Harness, benchmark, or regression planning               | Both agents                                                                 |
| Architecture: stage design, domain contracts             | `contextos-architect.agent.md`                                              |
| Authoring or workflow guidance                           | Both agents when planning is affected; implementer when editing is affected |
