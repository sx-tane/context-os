# ingest

Connector ingest and Codex re-auth orchestration helpers.

## Responsibilities

- Coordinate direct ingest, Codex-backed streaming ingest, cancellation, elapsed time, and component setter lifecycles.
- Keep connector components thin by centralizing API calls and error handling.
- Preserve abort behavior so user-initiated cancellation does not surface as a failure.

## Files

| File | Purpose |
| --- | --- |
| `runner.ts` | Runs direct or Codex connector ingest and forwards progress/result state to Svelte setters. |
| `reauthRunner.ts` | Runs Codex plugin re-auth streams and forwards log/status state to Svelte setters. |

## Maintenance Notes

- Follow `frontend-jest-swc-patterns` for tests because these helpers depend on mocked `$lib/api` calls.
- Run `bun run test` after lifecycle or error handling changes.
