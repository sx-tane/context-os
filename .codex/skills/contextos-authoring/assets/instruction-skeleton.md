# Instruction File Skeleton

Copy when creating a new `.instructions.md` file. Delete all `<!-- only if ... -->` annotations after applying.

---

## Template

```markdown
---
description: "Use when creating or modifying <X> in <location>. Covers <topic 1>, <topic 2>."
applyTo: "<glob>"
---

# <Subject> Instructions

## Skill <!-- only if a paired skill exists -->

For a full step-by-step guide, skeletons, and a completion checklist, apply the **<skill-name>** skill.

## <Concept> Shape

<Subject> must:

- <structural rule 1>
- <structural rule 2>
- <structural rule 3>

## <Operation> Checklist <!-- only if there is a repeatable operation (e.g. "New Route") -->

When adding a new <X>:

1. <step 1>
2. <step 2>
3. <step 3>
4. Run `go build ./...` and `go test ./...`.

## Do Not

- Do not <anti-pattern 1>.
- Do not <anti-pattern 2>.
```

---

## `applyTo` Glob Reference

| Files to cover                      | `applyTo` value                                   |
| ----------------------------------- | ------------------------------------------------- |
| All Go in API app                   | `apps/api/**/*.go`                                |
| All Go in source connectors         | `internal/source/**/*.go`                         |
| All Go in domain + internal + tests | `{domain,internal,tests}/**/*.go`                 |
| Reasoning / graph / presentation    | `internal/{reasoning,presentation,graph}/**/*.go` |
| Svelte components                   | `apps/frontend/**/*.svelte`                       |
| Python worker                       | `apps/ai-worker/**/*.py`                          |
| All Markdown                        | `**/*.md`                                         |

---

## Sizing Rules

Instructions are **loaded automatically** on every file match — every line costs context.

| Guideline                                | Target  |
| ---------------------------------------- | ------- |
| Total lines                              | < 50    |
| Sections                                 | ≤ 4     |
| Detail that belongs in a skill, not here | Move it |

---

## `description` Field — Writing Guide

The `description` is shown in the instruction picker and used by the agent to decide when to load the file on demand (if `applyTo` is absent).

**Pattern:**

```
"Use when creating or modifying <X>. Covers <rule 1>, <rule 2>."
```

**Anti-patterns:**

- Too long (> 200 chars) — gets truncated
- No "Use when" phrase — agent can't decide when to load it
- Replicates the `applyTo` glob in prose — redundant

---

## Existing Instructions in This Repo

| File                               | `applyTo`                                         | Paired skill                            |
| ---------------------------------- | ------------------------------------------------- | --------------------------------------- |
| `api-handlers.instructions.md`     | `apps/api/**/*.go`                                | `contextos-api-handler`                 |
| `connectors.instructions.md`       | `internal/source/**/*.go`                         | `contextos-api-handler`                 |
| `go-pipeline.instructions.md`      | `{domain,internal,tests}/**/*.go`                 | `go-best-practices`, `go-test-patterns` |
| `reasoning-output.instructions.md` | `internal/{reasoning,presentation,graph}/**/*.go` | —                                       |
