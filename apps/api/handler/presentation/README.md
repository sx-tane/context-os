# presentation handler

Routes under `/presentation/*` expose graph-backed role summaries for PMO, presentation layer, service layer, QA, and architecture views.

## Endpoints

| Method | Path                    | Description |
| ------ | ----------------------- | ----------- |
| GET    | `/presentation/status`  | Returns supported connectors/roles and hidden execution mode information. |
| POST   | `/presentation/findings` | Runs ingest + pipeline reasoning, then returns role-specific summaries, PMO view model, mismatches, and assistive execution evidence metadata. |

## Notes

- Execution evidence is assistive and never replaces deterministic mismatch evidence.
- The endpoint preserves mismatch IDs, confidence, impact, severity, evidence, and recommended next actions for API/UI stability.