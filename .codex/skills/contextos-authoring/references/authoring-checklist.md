# ContextOS Authoring — Completion Checklist

Run before marking any customization primitive complete.

---

## Skill Checklist

### Frontmatter

- [ ] `name` matches the folder name exactly (kebab-case, max 64 chars)
- [ ] `description` starts with a verb and includes "Use when:" with ≥ 2 trigger phrases
- [ ] `description` is ≤ 1024 chars
- [ ] `argument-hint` present if the skill takes a name/argument
- [ ] `user-invocable: true` unless the skill is agent-only

### Body

- [ ] `## Outcome` section lists all deliverable files with paths
- [ ] `## Decision Points` table covers branching conditions
- [ ] `## Procedure` uses numbered steps, each with a file path or skill reference
- [ ] At least one `assets/` skeleton file is referenced from a procedure step
- [ ] `references/` checklist file exists and is linked from the final procedure step
- [ ] `## Do Not` section lists ≥ 2 explicit anti-patterns

### Assets

- [ ] Every skeleton snippet has delete annotations (`<!-- only if ... -->`)
- [ ] Snippets compile / are syntactically valid
- [ ] No secrets, tokens, or environment-specific values in skeleton files

### Scripts

- [ ] `scripts/` exists only when the skill has a real executable helper
- [ ] Script paths and usage are referenced from `SKILL.md` or the checklist
- [ ] No empty script folders or placeholder scripts were added
- [ ] Authoring benchmark scripts pass after customization changes
- [ ] README coverage benchmark passes or exclusions are intentionally updated
- [ ] README quality benchmark passes for required directories
- [ ] README change-sync benchmark passes for the working diff range and rejects shallow documentation edits

### References

- [ ] Checklist has ≥ 10 numbered items grouped into sections
- [ ] Final section includes the relevant validation commands for that skill
- [ ] README update criterion is present for the nearest affected folder
- [ ] `.github/README.md` update criterion is present when customization routing changes

### Wiring

- [ ] Skill is referenced in `contextos-implementer.agent.md` (if implementation-facing)
- [ ] Skill is referenced in `contextos-architect.agent.md` (if architecture-facing)
- [ ] Each agent section has an inline fallback block (`If the skill is unavailable, enforce these inline rules:`)
- [ ] Skeleton link in agent section uses `../skills/<name>/assets/` relative path
- [ ] Checklist link in agent section uses `../skills/<name>/references/` relative path

---

## Instruction File Checklist

### Frontmatter

- [ ] `description` present with "Use when" phrase
- [ ] `applyTo` glob is as specific as possible (not `**`)
- [ ] YAML between `---` markers, no tab characters

### Body

- [ ] `## Skill` section present if a paired skill exists
- [ ] `## <Concept> Shape` section with ≥ 3 structural rules
- [ ] At most one `## <Operation> Checklist` section (≤ 6 steps)
- [ ] `## Do Not` section with ≥ 2 anti-patterns
- [ ] README alignment rule is present when the instruction changes editing behavior
- [ ] Total lines < 60

### No Cross-Duplication

- [ ] Deep implementation detail is in the skill, not repeated here
- [ ] Skeleton code is not duplicated in the instruction file

---

## Agent File Checklist

### Frontmatter

- [ ] `description` starts with "Use for" and names the agent's domain
- [ ] `name` is a human-readable display name (Title Case)
- [ ] `tools:` list contains only tools the agent actually needs
- [ ] `user-invocable: true`

### Body

- [ ] `## Mission` has ≥ 2 focused bullet points
- [ ] Every paired skill has its own `## <Feature> Delivery` section
- [ ] Each skill section has: skill name, skeleton link, checklist link, inline fallback rules
- [ ] `## Procedure` has numbered steps ending with tests + docs update
- [ ] `## Constraints` has ≥ 3 explicit guardrails
- [ ] `## Output` section lists the expected summary format
- [ ] `.github/README.md` stays aligned with agent skill wiring
- [ ] Mermaid explanation policy remains preserved for explanatory responses
- [ ] No instructions that conflict with `copilot-instructions.md` (architecture boundaries)

### Tool List Hygiene

- [ ] Read-only agents do not have `edit/*` or `execute/*` tools
- [ ] All `execute/*` tools included are justified by the agent's mission
- [ ] PR/issue tools only included if the agent is expected to open PRs or fetch issues
