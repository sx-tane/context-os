---
description: "Use when creating or modifying ContextOS agents, instruction files, or skills. Keeps customization routing, folder structure, and README documentation aligned."
applyTo: ".github/{agents,instructions,skills}/**/*.md,.codex/{agents,instructions,skills}/**/*.md"
---

# Customization Authoring Instructions

## Skill

For the full authoring workflow, skeletons, wiring rules, and completion checklist, apply the **contextos-authoring** skill.

## Customization Shape

- Skills live at `.github/skills/<name>/SKILL.md` and `.codex/skills/<name>/SKILL.md` mirrors; keep `assets/` and `references/` subdirectories aligned with the procedure.
- Add `scripts/` only when the skill has an executable helper; do not create empty script placeholders.
- Agents wire skills with a skill reference, skeleton link, checklist link, and inline fallback rules.
- Instruction files stay short and point to the paired skill for deep implementation detail.

## Documentation Checklist

1. Update `.codex/README.md`, `AGENTS.md`, and the relevant `.github/agents/README.md` or `.github/instructions/README.md` when agent, instruction, skill, routing, or folder structure behavior changes. Do not add a top-level `.github/README.md` only to satisfy benchmarks.
2. Update the nearest folder README when the customization changes how files in that folder should be edited.
3. Keep links to skeletons, checklists, examples, and scripts valid after every rename or move.

## Do Not

- Do not duplicate long skill procedures inside instruction files.
- Do not leave skill reference tables stale after adding or renaming a skill.
- Do not add unused script placeholders when no executable helper exists.
