# findings

Findings analysis orchestration and display view-model helpers.

## Responsibilities

- Run source-by-source analysis and aggregate successful findings without hiding per-source failures.
- Format findings, Activity rows, artifact source labels, source links, and safe chat line rendering.
- Keep display helpers deterministic and side-effect free except for the analysis runner.

## Files

| File | Purpose |
| --- | --- |
| `analysisRunner.ts` | Executes analysis for ready sources, handles source timeouts, and updates chat progress. |
| `aggregator.ts` | Combines per-source findings responses into one UI result. |
| `viewModel.ts` | Pure display helpers for findings, Activity, artifacts, and chat text lines. |

## Maintenance Notes

- Prefer metadata-backed source labels such as `source_label` for Activity grouping.
- Preserve backend JSON errors from analysis calls. For `source_too_broad`, show concrete repo, project, issue, channel, thread, document, or folder guidance with backend examples instead of a generic failure.
- Run `bun run test` after helper or analysis lifecycle changes.
