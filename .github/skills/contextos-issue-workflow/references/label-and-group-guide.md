# Label and Group Guide

Use this guide to keep issue groups consistent with existing ContextOS workflow patterns.

## Core Label Sets

Parent issue labels:
- `type: epic`
- `<area label>` (for example `area: connectors`)

Child issue labels:
- `type: feature`
- Same `<area label>` as parent

## Labels Used Across All Issues

Snapshot based on all current issues (`open` + `closed`) in `sx-tane/context-os`.

| Label | Usage count |
|-------|-------------|
| `type: feature` | 24 |
| `type: epic` | 6 |
| `area: connectors` | 12 |
| `area: intelligence` | 5 |
| `area: graph-reasoning` | 5 |
| `area: execution-validation` | 5 |
| `area: foundation` | 3 |

## Issue-to-Label Map

| Issue | Labels |
|------|--------|
| #33 Build Notion MCP connector | `type: feature`, `area: connectors` |
| #32 Build Confluence MCP connector | `type: feature`, `area: connectors` |
| #31 Build SharePoint / OneDrive MCP connector | `type: feature`, `area: connectors` |
| #30 Build Google Drive MCP connector | `type: feature`, `area: connectors` |
| #28 Main Group: Production Validation | `type: epic`, `area: execution-validation` |
| #27 Main Group: Outputs and Execution | `type: epic`, `area: graph-reasoning` |
| #26 Main Group: Context Graph and Reasoning | `type: epic`, `area: graph-reasoning` |
| #25 Main Group: Core Intelligence Pipeline | `type: epic`, `area: intelligence` |
| #24 Main Group: MCP Source Connectors | `type: epic`, `area: connectors` |
| #23 Issue 20: Validate first killer feature | `type: feature`, `area: execution-validation` |
| #22 Issue 19: Build presentation layer | `type: feature`, `area: execution-validation` |
| #21 Issue 18: Build hidden Codex execution integration | `type: feature`, `area: execution-validation` |
| #20 Issue 17: Build PMO summary output | `type: feature`, `area: execution-validation` |
| #19 Issue 16: Build mismatch detection | `type: feature`, `area: graph-reasoning` |
| #18 Issue 15: Build persistent context graph storage | `type: feature`, `area: graph-reasoning` |
| #17 Issue 14: Build relationship graph | `type: feature`, `area: graph-reasoning` |
| #16 Issue 13: Build identity resolution engine | `type: feature`, `area: intelligence` |
| #15 Issue 12: Build extraction engine | `type: feature`, `area: intelligence` |
| #14 Issue 11: Build classification engine | `type: feature`, `area: intelligence` |
| #13 Issue 10: Build normalization pipeline | `type: feature`, `area: intelligence` |
| #12 Issue 9: Build filesystem MCP connector | `type: feature`, `area: connectors` |
| #11 Issue 8: Build Excel MCP connector | `type: feature`, `area: connectors` |
| #10 Issue 7: Build OpenAPI MCP connector | `type: feature`, `area: connectors` |
| #9 Issue 6: Build Jira MCP connector | `type: feature`, `area: connectors` |
| #8 Issue 5: Build Slack MCP connector | `type: feature`, `area: connectors` |
| #7 Issue 4: Build GitHub MCP connector | `type: feature`, `area: connectors` |
| #6 Issue 3: Build MCP source connector interface | `type: feature`, `area: connectors` |
| #5 Issue 2: Define domain event contracts | `type: feature`, `area: foundation` |
| #4 Issue 1: Scaffold ContextOS repository structure (closed) | `type: feature`, `area: foundation` |
| #3 Main Group: Foundation and Contracts | `type: epic`, `area: foundation` |

## Grouping Rules

- Parent title should start with `Main Group:`.
- Every child issue must include `Part of #<parent-number>` in the body.
- Parent issue must list all child issues under `## Child issues`.
- Keep child issue language implementation-specific and testable.

## Validation Checklist

- Labels applied to parent and all children.
- Child list in parent is up to date.
- Parent reference in each child is correct.
- Production traceability section exists in every issue.