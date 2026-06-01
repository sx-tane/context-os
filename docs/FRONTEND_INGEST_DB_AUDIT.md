# Frontend Ingest And Database Audit

This audit traces what happens when the frontend installs project knowledge and whether the ingested data actually reaches PostgreSQL.

## Short Answer

The original frontend source setup flow was not a reliable database ingest flow.

The frontend could show a connector as `ready` after `/api/<connector>/ingest` or `/api/<connector>/ingest/stream` returned successfully, but those endpoints return events to the browser without persisting them to PostgreSQL.

The main `Install Knowledge` flow has now been changed to use `/api/presentation/findings`, because that handler calls the full pipeline with repository stores attached. The older connector debug forms still use the response-only ingest path and should not be treated as production-truthful until they are rewired too.

## Verified Runtime Behavior

The following was tested against the current source on `API_ADDR=:18080` with local Postgres running.

### 1. Workspace starts empty

Request:

```bash
curl -sS -X POST http://localhost:18080/workspace/upsert \
  -H 'Content-Type: application/json' \
  -d '{"path":"/tmp/contextos-current-audit","name":"current source audit"}'

curl -sS 'http://localhost:18080/workspace/status?path=%2Ftmp%2Fcontextos-current-audit'
```

Observed status:

```json
{
  "event_count": 0,
  "entity_count": 0,
  "mismatch_count": 0,
  "syncs": []
}
```

### 2. Direct ingest returns an event but does not persist

Request:

```bash
curl -sS -X POST http://localhost:18080/filesystem/ingest \
  -H 'Content-Type: application/json' \
  -d '{"uri":"inline-current.txt","content":"refund_status mismatch between API and UI"}'
```

Observed ingest response:

```json
{
  "connector": "filesystem",
  "event_count": 1
}
```

Observed workspace status immediately after:

```json
{
  "event_count": 0,
  "entity_count": 0,
  "mismatch_count": 0,
  "syncs": []
}
```

Conclusion: direct `/ingest` success does not mean the artifact is in the database.

### 3. Presentation findings does persist

Request:

```bash
curl -sS -X POST http://localhost:18080/presentation/findings \
  -H 'Content-Type: application/json' \
  -d '{
    "workspace_id":"/tmp/contextos-current-audit",
    "connector":"filesystem",
    "uri":"inline-current.txt",
    "content":"refund_status mismatch between API and UI",
    "role":"pmo",
    "include_execution":false,
    "force_refresh":true
  }'
```

Observed workspace status after:

```json
{
  "event_count": 1,
  "entity_count": 5,
  "mismatch_count": 1,
  "syncs": [
		{
		  "connector": "filesystem",
		  "source_uri": "inline-current.txt",
		  "event_count": 1,
		  "status": "idle"
		}
  ]
}
```

Observed artifacts query after:

```json
{
  "count": 1,
  "artifacts": [
    {
      "connector": "filesystem",
      "source_uri": "inline-current.txt",
      "title": "refund_status mismatch between API and UI"
    }
  ]
}
```

Conclusion: `/presentation/findings` runs the full pipeline with stores attached and writes events/entities/mismatches to PostgreSQL.

## Code Path Analysis

### Frontend source setup path

`apps/frontend/src/lib/components/knowledge/KnowledgeInstall.svelte`

- User enables a connector and enters a URI.
- `installAll()` calls `runConnectorIngest()`.
- On successful `setResult`, it calls `setConnectorKnowledge(..., "ready")`.
- That ready state is stored in frontend `localStorage`, not guaranteed by database state.

`apps/frontend/src/lib/ingestRunner.ts`

- For Codex providers, calls `streamCodexIngest()`.
- For direct providers, calls `postIngest()`.

`apps/frontend/src/lib/api.ts`

- `postIngest()` calls `POST /api/<connector>/ingest`.
- `streamCodexIngest()` calls `POST /api/<connector>/ingest/stream`.
- Neither request includes a `workspace_id`.

### API direct ingest path

`apps/api/handler/shared/ingest.go`

- `RunSourceIngest()` decodes the request.
- `WriteSourceIngest()` calls `connector.Ingest(...)`.
- It returns `response.Ingest`.
- It does not call `pipeline.Run(...)`.
- It does not receive repository stores.
- It does not write to `EventStore`, `EntityStore`, `MismatchStore`, or `SyncStore`.

This means all connector handlers using this shared path are response-only unless they have custom persistence elsewhere.

### API persistence path

`apps/api/handler/presentation/presentation.go`

- Stateful handler is created in `apps/api/main.go` only when DB connection succeeds.
- `Findings()` ensures the workspace exists.
- It builds `pipeline.Stores`.
- It calls `pipeline.Run(..., stores)`.

`internal/pipeline/pipeline.go`

- `Run()` ingests source events.
- It normalizes, classifies, extracts, resolves identity, builds relationships, and detects mismatches.
- If stores are present, `persistResult()` writes:
  - ingest events to `ingest_events`
  - entities to `entities`
  - relationships to `relationships`
  - mismatches to `mismatches`
  - graph snapshots to `storage/snapshots`

## Frontend User-Visible Consequences

The UI can currently enter a misleading state:

- Source setup can mark a connector as `ready`.
- Workspace metrics can still show `0 local events`.
- Chat can answer `No local artifacts were found` even after an ingest that visually looked successful.
- Graph can remain empty after "Install Knowledge".
- Running analysis may later populate the DB, which makes it feel inconsistent.

This is not a styling issue. It is a product/data-flow issue.

## Additional Issues Found

### Direct ingest has no workspace scope

The direct ingest API request shape does not include `workspace_id`, so the API has no place to persist the event even if it wanted to.

### Codex stream ingest likely has the same persistence gap

The frontend Codex path calls `/ingest/stream`, which returns a final ingest payload over SSE. That path follows the same response-oriented pattern and does not expose workspace-scoped store writes in the frontend request.

### Connector-level sync count was corrected

`/presentation/findings` now returns the pipeline `event_count` and writes the same count into `connector_syncs.event_count`. This makes the workspace total event count and the per-connector sync count agree for the DB-backed findings pipeline.

### LocalStorage is treated as source readiness

Frontend project state stores connector readiness in `localStorage`. This is useful for UI continuity, but it should be reconciled against `/workspace/status` before claiming a source is truly ready.

## Production Fix Order

### Phase A: Make Ingest Persist

Add workspace scope to ingest requests:

- `workspace_id` or `workspace_path`
- connector
- URI
- provider/token/content/cursor/metadata

Then change ingest handlers so they can run the full pipeline with stores attached, or introduce a dedicated persistent ingest service used by both `/ingest` and `/presentation/findings`.

Exit criteria:

- Calling `/filesystem/ingest` with `workspace_id` increases `/workspace/status.event_count`.
- Calling `/artifacts?workspace_id=...` returns the ingested artifact.
- Repeating the same ingest does not duplicate the event.

### Phase B: Fix Frontend Source Setup Truth

Change `KnowledgeInstall` so "ready" means persisted:

- Send workspace path/id with every ingest.
- After ingest, call `/workspace/status`.
- Mark connector `ready` only when the DB has a matching sync/artifact count.
- Show "returned but not persisted" as an error state if status does not change.

Exit criteria:

- Source setup cannot show ready while workspace event count remains unchanged.
- Chat can find artifacts immediately after source setup.
- Graph count updates after persisted pipeline output.

### Phase C: Keep Sync Counts Correct

When pipeline persistence succeeds, keep writing the real persisted event count to `connector_syncs.event_count`.

Exit criteria:

- Workspace total event count and per-connector sync event count agree for a single-source workspace.

### Phase D: Add Regression Tests

Add API tests for persistent ingest:

- direct filesystem ingest persists event
- repeat ingest is idempotent
- artifact query sees persisted event
- frontend ingest runner sends workspace scope

Exit criteria:

- A future change cannot reintroduce the "ready but not in DB" bug.

## Current Recommendation

Treat the main `Install Knowledge` flow as the current DB-backed source setup path. Do not treat connector debug pages or raw `/ingest` calls as persisted ingest until they are updated to send workspace scope and use the same persistence pipeline.
