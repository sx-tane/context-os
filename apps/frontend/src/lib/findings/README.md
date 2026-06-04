# findings

Findings analysis orchestration and display view-model helpers.

## Responsibilities

- Run source-by-source analysis and aggregate successful findings without hiding per-source failures.
- Use selected analysis basket evidence first, then concrete chat/Activity evidence, as analysis inputs when Sources contains only broad live connector scopes.
- Format findings, Activity rows, artifact source labels, source links, safe chat line rendering, and markdown-preserving detail text.
- Keep display helpers deterministic and side-effect free except for the analysis runner.

## Files

| File | Purpose |
| --- | --- |
| `analysisRunner.ts` | Executes analysis for selected basket sources, concrete ready sources, and chat/Activity-derived evidence, applies provider-specific source timeouts, and updates chat progress. |
| `aggregator.ts` | Combines per-source findings responses into one UI result. |
| `viewModel.ts` | Pure display helpers for findings, latest findings runs from chat cards, Activity, artifacts, markdown-safe detail text, and chat text lines. |

## Maintenance Notes

- Prefer metadata-backed source labels such as `source_label` for Activity grouping.
- Preserve backend JSON errors from analysis calls. For `source_too_broad`, show concrete repo, project, issue, channel, thread, document, or folder guidance with backend examples instead of a generic failure.
- Keep `/presentation/findings` requests concrete. Selected basket items override the derived source list for a run; otherwise broad Sources rows are reported as chat-only scopes while concrete `answer_sections` and Activity artifacts can supply analysis targets without mutating the Sources setup list.
- Keep Codex-backed source timeouts aligned with the API findings timeout: plugin-backed sources get 5 minutes per source, while direct/token sources keep the shorter 90 second frontend guard.
- Run `bun run test` after helper or analysis lifecycle changes.
