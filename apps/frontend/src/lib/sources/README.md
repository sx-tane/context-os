# sources

Shared source helpers for setup and analysis routing.

## Modules

| File | Purpose |
| --- | --- |
| `analysisEligibility.ts` | Preserves typed source URIs during setup and splits ready sources into concrete findings-analysis inputs versus chat-only live connector scopes. |

## Behavior

Connector-only live scopes such as `github`, `jira`, `slack`, or `googledrive` remain valid saved sources for chat because the live plugin can search the connected account. Findings analysis requires concrete scope, so only filesystem sources and plugin-backed sources with a specific repo, project, issue, channel, document, folder, or file are eligible for `/presentation/findings`.

Keep these helpers deterministic and UI-independent so setup components, route summaries, and tests share the same source-scope rules.
