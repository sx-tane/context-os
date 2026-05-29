# Agent Skeleton

Copy when creating a new `.agent.md` for ContextOS. Delete all `<!-- only if ... -->` annotations after applying.

---

## Template

```markdown
---
description: "Use for <primary role>. Best suited for <type of task>."
name: "<Display Name>"
tools: [
    vscode/extensions,
    vscode/askQuestions,
    vscode/memory,
    vscode/resolveMemoryFileUri,
    vscode/runCommand,
    vscode/vscodeAPI,
    vscode/toolSearch,
    execute/runInTerminal,
    <!-- only if agent needs to run commands -->
    execute/runTests,
    <!-- only if agent runs tests -->
    execute/getTerminalOutput,
    <!-- only if agent reads terminal output -->
    execute/killTerminal,
    <!-- only if agent manages terminal lifecycles -->
    execute/sendToTerminal,
    <!-- only if agent sends input to terminals -->
    execute/runTask,
    <!-- only if agent runs VS Code tasks -->
    execute/createAndRunTask,
    <!-- only if agent creates + runs VS Code tasks -->
    execute/testFailure,
    <!-- only if agent inspects test failures -->
    read/terminalSelection,
    read/terminalLastCommand,
    read/problems,
    read/readFile,
    read/viewImage,
    search/codebase,
    search/fileSearch,
    search/listDirectory,
    search/textSearch,
    search/usages,
    agent/runSubagent,
    <!-- only if agent spawns sub-agents -->
    edit/createDirectory,
    <!-- only if agent creates directories -->
    edit/createFile,
    <!-- only if agent creates files -->
    edit/editFiles,
    <!-- only if agent edits files -->
    edit/rename,
    <!-- only if agent renames files -->
    todo,
    web/githubRepo,
    web/githubTextSearch,
    github.vscode-pull-request-github/issue_fetch,
    <!-- only if agent works with issues/PRs -->
    github.vscode-pull-request-github/create_pull_request,
    <!-- only if agent creates PRs -->,
  ]
user-invocable: true
---

You are a ContextOS <role> specialist.

## Mission

- <primary mission statement 1>.
- <primary mission statement 2>.

## <Primary Skill / Domain> <!-- one section per skill this agent applies -->

When <doing X>, apply the **<skill-name>** skill.
Use the [skeleton](../skills/<skill-name>/assets/<x>-skeleton.md) for new files.
Run the [checklist](../skills/<skill-name>/references/<x>-checklist.md) before marking complete.
If the skill is unavailable, enforce these inline rules:

- <inline fallback rule 1>
- <inline fallback rule 2>

## Procedure

1. <step 1>
2. <step 2>
3. Run tests: `go test ./...` and `go vet ./...` on all touched packages.
4. Update all relevant `.md` files affected by the change.

## Constraints

- <guardrail 1 — what must never happen>.
- <guardrail 2 — what must never happen>.
- If a change modifies an exported contract, document the breaking-change impact before editing callers.
- Whenever updating code, always update relevant Markdown documentation in the same change.

## Output

- Files changed and purpose
- Behavior change summary
- Markdown documentation updated
- Test results
- Remaining risk or debt
```

---

## Tools Reference

Only include tools the agent actually needs. Extra tools slow down discovery.

### Read-only agents (research/planning)

Keep only `read/*`, `search/*`, `vscode/*`, `web/*`, `todo`.

### Implementation agents

Add `edit/*`, `execute/runInTerminal`, `execute/runTests`, `agent/runSubagent`.

### PR/issue agents

Add `github.vscode-pull-request-github/*` tools.

---

## Inline Fallback Rules

Every `## <Feature> Delivery` section must have inline fallback rules.
These are shown to the model when the skill file itself cannot be loaded.
Keep fallback rules to 4–6 bullets covering the most critical structural requirements.

---

## Existing Agents in This Repo

| File                             | Display Name          | Role                                       |
| -------------------------------- | --------------------- | ------------------------------------------ |
| `contextos-implementer.agent.md` | ContextOS Implementer | Pipeline implementation, connectors, tests |
| `contextos-architect.agent.md`   | ContextOS Architect   | Architecture planning, stage design        |
