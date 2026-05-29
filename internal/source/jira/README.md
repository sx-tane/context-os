# Jira Source

Source connector for Jira issue and project artifacts, with the UI/API defaulting to Codex Rovo plugin ingestion.

## What It Does

- Exposes the public connector name `jira` with `issues` capability.
- Accepts Jira browse URLs, `jira://issue/<KEY>`, and `jira://project/<KEY>`.
- Enriches metadata with issue key, project key, host, object type, object ID, and stable `source_id`.
- Supports direct Jira REST reads when token/env auth is configured.
- The Codex provider path uses `atlassian-rovo@openai-curated` and is wired through the API/UI.

## Important Files

| File           | Role                                                               |
| -------------- | ------------------------------------------------------------------ |
| `jira.go`      | Direct Jira API connector, URI parsing, auth, metadata enrichment. |
| `jira_test.go` | Jira URI, auth, status, cursor, and error behavior tests.          |

## Boundary

Do not add a separate Confluence source package for Atlassian context right now. Atlassian plugin-backed context should route through Codex/Rovo unless a dedicated issue reopens that scope.
