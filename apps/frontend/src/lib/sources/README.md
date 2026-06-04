# sources

Shared source helpers for setup and analysis routing.

## Modules

| File | Purpose |
| --- | --- |
| `analysisEligibility.ts` | Preserves typed source URIs during setup, splits ready sources into concrete findings-analysis inputs versus chat-only live connector scopes, and derives concrete analysis inputs from saved chat/Activity evidence. |

## Behavior

Connector-only live scopes such as `github`, `jira`, `slack`, or `googledrive` remain valid saved sources for chat because the live plugin can search the connected account. Findings analysis requires concrete scope, so only filesystem sources and plugin-backed sources with a specific repo, project, issue, channel, document, folder, or file are eligible for `/presentation/findings`.

When chat or Activity already contains concrete evidence, `buildAnalysisSources` merges it with ready Sources rows before analysis runs. It reads `answer_sections` and saved artifacts with `connector + source_uri`, falls back to concrete section links when a source URI is missing, deduplicates by `connector + uri`, skips broad connector scopes, and caps derived evidence to the same 12-item safe source-card limit used by chat evidence.

Keep these helpers deterministic and UI-independent so setup components, route summaries, and tests share the same source-scope rules.
