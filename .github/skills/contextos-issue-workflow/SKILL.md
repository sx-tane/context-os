---
name: contextos-issue-workflow
description: "Create or update ContextOS parent-child GitHub issues using the established Main Group and child workflow, including labels, group linkage, and production traceability sections."
argument-hint: "What feature area should be grouped and what child issues are needed?"
user-invocable: true
---

# ContextOS Issue Workflow

## Outcome

Create or update a parent group issue and aligned child issues that follow ContextOS format, labels, and production traceability requirements.

## When to Use

- Creating a new implementation wave that spans multiple connector or pipeline tasks.
- Converting ad-hoc issues into a tracked parent-child group.
- Standardizing issue bodies to match existing ContextOS issue format.

## Required Format

- Parent title format: `Main Group: <theme>`.
- Child title format: action-oriented task title (for example: `Build <Source> MCP connector`).
- Parent body includes: Direction, Child issues, Done when.
- Child body includes: Description, Parent issue, Direction, Acceptance criteria.
- Both parent and child include the production traceability block.

## Label and Group Rules

- Parent labels: `type: epic` and area label (for example `area: connectors`).
- Child labels: `type: feature` and the same area label as parent.
- Parent-child link must appear in both directions:
  - Child body includes `Part of #<parent>`.
  - Parent body includes the child issue list.

## Procedure

1. Define the feature theme and choose the area label.
2. Create the parent issue from [parent-issue-template.md](./assets/parent-issue-template.md).
3. Create child issues from [child-issue-template.md](./assets/child-issue-template.md).
4. Apply label sets from [label-and-group-guide.md](./references/label-and-group-guide.md).
5. Add child issue numbers into the parent `Child issues` list.
6. Validate link integrity and acceptance coverage.

## Completion Checks

- Parent and child issues use the expected section ordering.
- Labels are present and consistent across the group.
- Parent-child links are complete and accurate.
- Production traceability blocks are present in every issue.
- Acceptance criteria are implementation-testable.

## References

- [Parent Issue Template](./assets/parent-issue-template.md)
- [Child Issue Template](./assets/child-issue-template.md)
- [Label and Group Guide](./references/label-and-group-guide.md)
- [Issue Creation Script](./scripts/create-parent-child-issues.sh)
