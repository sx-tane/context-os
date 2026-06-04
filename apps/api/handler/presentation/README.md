# presentation handler

Routes under `/presentation/*` expose graph-backed role summaries for PMO, presentation layer, service layer, QA, and architecture views.

## Endpoints

| Method | Path                    | Description |
| ------ | ----------------------- | ----------- |
| GET    | `/presentation/status`  | Returns supported connectors/roles and hidden execution mode information. |
| POST   | `/presentation/findings` | Runs ingest + pipeline reasoning, then returns role-specific summaries, PMO view model, mismatches, and assistive execution evidence metadata. |

## Notes

- Execution evidence is assistive and never replaces deterministic mismatch evidence.
- The handler delegates analysis through the grouped stage packages under `internal/stages/*` plus source connectors under `internal/source/*`; route behavior stays the same even though internal import paths are grouped.
- The endpoint preserves mismatch IDs, confidence, impact, severity, evidence, and recommended next actions for API/UI stability.
- Codex-backed findings reject connector-only URIs such as `github`, `jira`, `slack`, `googledrive`, `notion`, and `sharepoint` with `400 {"error":"source_too_broad"}` plus concrete examples. Users must choose a specific repo, project, issue, channel, thread, document, or folder before local analysis runs.
- Fresh findings responses include `entity_count` and `relationship_count` from the pipeline run so the frontend can explain the resulting graph size.
- Sync cursor and audit writes use a detached 30 second `presentationWriteTimeout` context so client cancellation does not interrupt operational persistence after findings complete.
