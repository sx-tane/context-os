# Skill Skeleton

Copy this skeleton when creating a new ContextOS skill. Delete all `<!-- only if ... -->` annotations after applying.

---

## 1. Folder Structure

```
.github/skills/<name>/
├── SKILL.md
├── assets/
│   └── <name>-skeleton.md      <!-- only if the skill needs code/template assets -->
├── references/
│   └── <name>-checklist.md
└── scripts/                    <!-- only if the skill has an executable helper -->
   └── <helper>.sh             <!-- only if a helper is useful -->
```

---

## 2. SKILL.md Template

```markdown
---
name: <name>
description: "Create / implement <X> following the established ContextOS pattern. Use when: <trigger phrase 1>; <trigger phrase 2>; <trigger phrase 3>. Covers <topic 1>, <topic 2>, and <topic 3>."
argument-hint: "What is the <subject> name?"   <!-- only if user will supply a name argument -->
user-invocable: true
---

# ContextOS <Name> Skill

## Outcome

Deliver a fully-wired, tested <X> with:

- `<path/to/file>` — <what it contains>
- `<path/to/file>` — <what it contains>

---

## Decision Points <!-- only if branching logic is needed -->

| Situation   | Action       |
| ----------- | ------------ |
| <condition> | <what to do> |
| <condition> | <what to do> |

---

## Procedure

1. **<Step name>** — <description>.
   See pattern in [skeleton](./assets/<name>-skeleton.md).

2. **<Step name>** — <description>.
   Apply the **<dependency-skill-name>** skill.

3. **Document** — update the nearest README affected by the skill.

4. **Validate** — run the [checklist](./references/<name>-checklist.md) before marking complete.

---

## Do Not

- Do not <anti-pattern 1>.
- Do not <anti-pattern 2>.
```

---

## 3. Checklist Template (`references/<name>-checklist.md`)

See [authoring-checklist.md](../references/authoring-checklist.md) for the meta-checklist.

For the skill-specific checklist, use this pattern:

```markdown
# <Name> Skill — Completion Checklist

Run before marking the task complete.

## <Section A>

- [ ] <criterion 1>
- [ ] <criterion 2>

## <Section B>

- [ ] <criterion 3>
- [ ] <criterion 4>

## Final

- [ ] Relevant validation command passes
- [ ] Nearest `README.md` updated when behavior, commands, structure, or routing changed
- [ ] `.github/README.md` updated when the skill itself changes customization routing
```

---

## 4. Description Field — Writing Guide

The `description` field is the **only thing the agent reads at discovery time** (~100 tokens). Every trigger phrase that should cause the agent to load this skill MUST appear in the description.

**Pattern:**

```
"<Verb> a <Subject> following the established <Repo> pattern.
Use when: <trigger 1>; <trigger 2>; <trigger 3>.
Covers <concept 1>, <concept 2>, <concept 3>."
```

**Bad** (too vague — agent won't find it):

```
description: "Helps with API stuff."
```

**Good** (specific trigger phrases):

```
description: "Create a new ContextOS API handler and its paired source connector. Use when: adding a /connector/status route; adding a /connector/ingest route; adding an internal/source/<name> connector."
```
