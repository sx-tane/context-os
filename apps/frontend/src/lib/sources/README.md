# sources

Shared source helpers for setup and analysis routing.

## Modules

| File | Purpose |
| --- | --- |
| `analysisEligibility.ts` | Preserves typed source URIs during setup, splits ready sources into concrete findings-analysis inputs versus chat-only live connector scopes, prioritizes selected basket evidence, and derives concrete analysis inputs from saved chat/Activity evidence. |

## Behavior

Connector-only live scopes such as `github`, `jira`, `slack`, or `googledrive` remain saved setup/status rows and can answer chat as broad connected-account lookups. They are still not analysis-ready evidence by themselves: Findings analysis requires a concrete scope such as a repo, project, issue, channel, document, folder, or file, and chat evidence saves only concrete provenance returned by the live answer.

When the analysis basket contains concrete evidence, `buildAnalysisSources` uses those selected items as the eligible run list and keeps the automatically available sources separately for status display. Without basket selections, chat or Activity evidence is merged with ready Sources rows before analysis runs. The helper reads `answer_sections`, saved artifacts with `connector + source_uri`, and basket items with `connector + uri`; it falls back to concrete section links when a source URI is missing, deduplicates by `connector + uri`, skips broad connector scopes, and caps derived evidence to the same 12-item safe source-card limit used by chat evidence.

Keep these helpers deterministic and UI-independent so setup components, route summaries, and tests share the same source-scope rules.
