# lib

Shared TypeScript modules for the ContextOS frontend. Routes and components import these modules through the `$lib` alias.

## Module Map

| Path | Purpose |
| --- | --- |
| [`api/`](api/) | HTTP/SSE API client, request ID correlation, and quiet-by-default browser request logging. `$lib/api` remains the public API-client entrypoint. |
| [`workspace/`](workspace/) | Svelte stores for workspace project state, local chat history, selected connectors, protected demo/default workspaces, and backend workspace registration. |
| [`chat/`](chat/) | Chat command orchestration plus protected demo workspace seed data. |
| [`findings/`](findings/) | Findings analysis runner, per-source result aggregation, and display-only findings/chat formatting helpers. |
| [`insights/`](insights/) | Shared Graph/Findings/Activity freshness and source-eligibility status helpers for the homepage insight surface. |
| [`graph/`](graph/) | Display-only graph view model helpers for focused entity maps and relationship details. |
| [`ingest/`](ingest/) | Connector ingest and Codex plugin re-auth orchestration helpers. |
| [`connectors/`](connectors/) | Static source connector configuration consumed by connector routes. |
| [`components/`](components/) | Svelte UI components grouped by responsibility. |
| [`generated/`](generated/) | Auto-generated OpenAPI TypeScript declarations. Do not edit by hand. |
| [`types.ts`](types.ts) | Canonical frontend type definitions and generated API type re-exports. |

Keep route and component files thin: put network behavior in `api/`, workspace persistence in `workspace/`, chat flow in `chat/`, analysis/report shaping in `findings/`, insight freshness in `insights/`, graph display shaping in `graph/`, and connector lifecycle orchestration in `ingest/`.

---

## api/

Single source of truth for all communication with the Go API (`/api`).

**Public entrypoint:** `$lib/api`

**Exports**

| Symbol | What it does |
| --- | --- |
| `API_URL` | Base path constant (`/api`). |
| `apiFetch(input, init)` | Shared fetch wrapper that adds `X-ContextOS-Request-ID`; optional timing logs are handled by `api/logger.ts`. |
| `probeService(url)` | `GET /health` with a 3 s timeout; returns `"ok"` or `"unreachable"`. |
| `getJSON<T>(path)` | Generic `GET` that deserialises JSON or returns `null` on any error. |
| `postIngest(connector, body, opts)` | `POST /<connector>/ingest` with a JSON body; returns `{ ok: true, body: IngestResult }` or `{ ok: false, body: ApiErrorBody }`. |
| `postFindings(body, opts)` | `POST /presentation/findings`; preserves backend JSON errors and converts network fetch failures into `{ ok: false, status: 0, body: { error: "api_unreachable", message } }`. |
| `postFilesystemUpload(formData, opts)` | `POST /filesystem/upload` with `multipart/form-data`; callers include `workspace_id` plus browser-selected `files` and folder-relative `paths` so local uploads are copied into ContextOS storage and ingested locally. |
| `getCodexSources(connector)` | `GET /codex/sources?connector=...` for Codex-discoverable live sources: GitHub repos, Jira projects, Slack channels, Notion pages/databases, Google Drive folders/docs, and SharePoint/OneDrive locations. |
| `getWorkspaces()` | Fetches registered API workspaces and returns an empty list when unavailable. |
| `upsertWorkspace(path, name)` | Registers or updates a local workspace path. |
| `postWorkspaceSource(body, opts)` | Saves an external connector/source URI as a connected source reference through `POST /workspace/source`; this does not ingest content. |
| `deleteWorkspace(path)` | Calls `DELETE /workspace?path=...` and returns structured `{ ok, status, message? }` details so the route can remove local state while reporting backend/API failures. |
| `getWorkspaceStatus(path)` | Fetches workspace event/entity/mismatch counts and connector sync state. |
| `getArtifacts(params)` | Queries local source artifacts from `GET /artifacts` by workspace, connector, source URI, date range, text, and limit. |
| `cleanupLiveEvidence(workspaceID)` | Calls `POST /artifacts/live-evidence/cleanup` and returns deleted noisy live-evidence counts for explicit Activity cleanup. |
| `cleanupGraphNoise(workspaceID)` | Calls `POST /graph/cleanup` and returns matched/deleted counts for explicit permanent graph cleanup; it removes low-signal graph rows only, not artifacts, findings, chat history, or connected sources. |
| `postChatQuery(body, opts)` | Sends chat questions to `POST /chat/query`; plugin-backed concrete source links use live Codex first, start eligible Local DB live-answer evidence saves asynchronously, then fall back to local artifacts when needed. Network failures return a structured `api_unreachable` error instead of throwing raw `Failed to fetch`. |
| `streamChatQuery(body, handlers, opts)` | Opens an SSE stream to `POST /chat/query/stream`; dispatches live Codex `onLog`, heartbeat `onStatus`, early `onAnswer`, final `onResult` with evidence-save status, and `onError` events for the pending chat transcript. |
| `streamCodexIngest(connector, body, handlers, opts)` | Opens an SSE stream to `POST /<connector>/ingest` for Codex-backed connectors; dispatches `onLog`, `onStatus`, `onResult`, and `onError` events. |
| `streamCodexReauth(plugin, onLog, opts)` | Opens an SSE stream to `POST /codex/plugin-reauth?plugin=...`; forwards each log line to `onLog`. |
| `streamCodexLogin(onLog, opts)` | Opens an SSE stream to `POST /codex/login`; forwards each log line to `onLog`. |

All streaming functions use a shared `readEventStream` helper that parses SSE blocks (`event: ...\ndata: ...\n\n`) from the response body reader.

### api/logger.ts

Centralizes frontend request correlation. API calls receive an `X-ContextOS-Request-ID` header even when logging is disabled. Browser console request logs are quiet by default; enable them with `contextosAPITrace(true)` in the browser console, `localStorage.contextos_debug_api = "1"`, or `VITE_CONTEXTOS_DEBUG_LOGS=1`. Match the browser `id=web-...` value to API terminal request logs when `CONTEXTOS_API_REQUEST_LOGS=1` is enabled.

---

## workspace/projectStore.ts

Maintains local workspace state in browser storage and registers user-created workspaces with the backend. The store keeps `status="connected"` and `status="pending"` sync rows ready without assigning empty ingest event counts, because those rows represent saved external source references that can still be used by live chat or future sync. Backend `error` rows remain visible as connector errors. The store always exposes the default workspace and a protected `contextos-demo` workspace; neither protected workspace is marked removed or deleted from local storage. The demo workspace is local-only and is not registered with the backend when opened.

`workspace/statusMapping.ts` owns the pure reconciliation logic so backend sync-state mapping can be tested without importing Svelte stores or Vite runtime globals.

Cached project and chat data is bounded on both load and save: old browser state is trimmed to 200 chat messages, 80 stream lines per message, 20 cached evidence artifacts per chat card, 100 workspaces, and 100 connectors per workspace.

---

## chat/

### chat/controller.ts

Owns homepage chat command routing and source query execution. The route passes callbacks for Svelte state/store updates, while this module handles command classification, chat message construction, demo query answers, backend `streamChatQuery`, non-streaming `postChatQuery` fallback, and source-query error messages.

The loading stream intentionally names both live Codex sources and the local DB because plugin-backed source questions ask live context first, then use persisted artifacts as fallback and evidence history. `runChatQuery` reuses that inferred route for the request body, so concrete prompts such as `BKGDEV-8466 check this` send `connector: "jira"` and `source_uri: "BKGDEV-8466"` to both stream and fallback routes. Spreadsheet filename prompts such as `BKGDEV-8096_帳票項目のマッピング確認.xlsx ... Jira ... Slack` send `connector: "googledrive"` without forcing a Jira source URI, letting the backend save concrete Drive/Jira/Slack provenance from the returned answer. Chat requests include a deterministic `response_language` hint (`zh`, `ja`, `ko`, or `en`) so live Codex answers and deterministic local fallback/status answers match the user's input language.

During `/chat/query/stream`, Codex-style `>` and `*` progress lines update `ChatMessage.stream` while `ChatMessage.text` remains reserved for the answer. Final streamed results attach the `ChatQueryResult`, mark the stream complete, and keep a compact Local DB save summary such as `Local DB: saved 8 artifacts; graph updated` for the chat panel. Structured `answer_sections` are preserved on cached chat cards and rendered as source cards by the chat panel. If the stream fails after an early answer, the answer remains visible and the UI reports the Local DB save failure instead of marking the live lookup failed. Saved evidence refreshes workspace Activity and Graph immediately; Findings stay unchanged until analysis runs.

### chat/demoWorkspace.ts

Provides the protected `contextos-demo` workspace records used by the homepage when users want to inspect the intended findings, graph, activity, and chat experience without connecting live sources.

---

## findings/

### findings/analysisRunner.ts

Owns the homepage analysis execution loop. It runs concrete ready sources one at a time, chooses direct token vs Codex provider, updates progress messages, aggregates successful findings, preserves per-source failures, and reports a clear zero-finding result when analysis completes without mismatch signals. Connector-only live scopes such as `github:github` remain chat-ready but are skipped before findings analysis because the backend requires a concrete repo, project, issue, channel, thread, document, folder, or file.

### findings/aggregator.ts

Merges per-source `postFindings` responses into one `FindingsResult` for the homepage. It combines mismatch arrays, sums mismatch/event/entity counts, and builds the chat summary text that distinguishes a successful zero-finding analysis from source failures.

### findings/viewModel.ts

Keeps presentation-only formatting outside the route and insight components: severity labels, finding text fallbacks, message line parsing, artifact origin/provider labels, artifact source link extraction, Activity event summaries, preview truncation, and timestamp formatting. Chat line parsing preserves Japanese and other non-English content; inline Markdown rendering is handled by the chat components without raw HTML injection. Connector labels such as Jira, Slack, GitHub, Google Drive, Notion, SharePoint, and Filesystem are promoted into subtle section rows so long answers read as grouped report sections without changing the chat layout.

---

## insights/

### insights/status.ts

Derives one shared freshness/status model for the homepage insight surface. `buildInsightStatus` counts concrete analysis-ready sources separately from chat-only live connector scopes, finds the latest Activity evidence timestamp, summarizes Graph node/link availability, and classifies manual Findings as `not_run`, `current`, `stale`, or `no_concrete_sources`. The route uses the same derived labels for the insight status strip and footer, while `FindingsView` uses the state-specific copy to explain why Findings may differ from fresher Activity or Graph evidence.

---

## graph/viewModel.ts

Builds the focused graph model consumed by `GraphView.svelte`: explicit or inferred links, entity degrees, selected/linked/top index sections, focus rows, relationship groups, stable type colors, and readable relationship labels.

---

## ingest/

### ingest/runner.ts

Coordinates a complete ingest lifecycle for a single connector so components stay thin.

**Exported function**

```ts
runConnectorIngest(options: IngestRunnerOptions): Promise<void>
```

`IngestRunnerOptions` carries:

| Field | Purpose |
| --- | --- |
| `connector` | Which `ConnectorKind` to call. |
| `workspace_id` | Optional workspace path or ID forwarded to direct and Codex ingest requests so backend persistence is workspace-scoped. |
| `uri`, `token`, `content`, `cursor`, `metadata` | Ingest payload fields. |
| `provider` | `"codex"` routes through `streamCodexIngest`; anything else uses `postIngest`. |
| `setLoading`, `setError`, `setResult`, `setLiveLog`, `setElapsed` | Svelte reactive setters the runner calls as state changes. |
| `isCurrent()` | Optional guard: if it returns `false`, all setter calls are silently dropped to prevent stale updates after a component switch. |
| `signal` | `AbortSignal` for cancellation; abort errors are silently swallowed. |

**Flow**

```text
provider === "codex"  ->  streamCodexIngest  ->  onLog / onStatus / onResult / onError
provider !== "codex"  ->  postIngest         ->  result or error body
```

A `setInterval` timer increments `elapsed` every second during Codex streaming and is cleared in `finally`.

Connector components should pass the current `$project.workspacePath` as `workspace_id` so all ingest providers write to the active workspace.

### ingest/reauthRunner.ts

Wrapper around `streamCodexReauth` for the Codex plugin re-auth flow.

> **Not currently wired into the UI.** To reconnect a plugin to a different account, run `codex plugin remove <plugin>@openai-curated && codex plugin add <plugin>@openai-curated` directly in your terminal. Tracked in issue: **Add frontend Codex plugin re-auth flow**.

**Exported function**

```ts
runCodexReauth(options: ReauthRunnerOptions): Promise<void>
```

`ReauthRunnerOptions` carries:

| Field | Purpose |
| --- | --- |
| `plugin` | Plugin name passed to the re-auth endpoint. |
| `refreshCodexStatus()` | Called in `finally` to reload Codex status after the stream ends. |
| `setPlugin`, `setLog`, `setRunning` | Svelte reactive setters. |
| `isCurrent()` | Same stale-update guard as `IngestRunnerOptions`. |
| `signal` | `AbortSignal`; abort errors are silently swallowed. |

---

## connectors/sourceConnectorConfigs.ts

Static array of `SourceConnectorConfig` objects. One entry per `DirectSourceConnectorKind`.

Each config is consumed by the route page to render a `SourceConnector` card without hardcoding connector details in the UI. Currently contains a single entry for the filesystem connector. Add a new object here when a new direct non-Codex connector is introduced.

---

## types.ts

Central type registry for the frontend.

**Auto-generated from swagger** (do not edit these directly; run `bun run codegen` to refresh):

| Type alias | Generated source |
| --- | --- |
| `IngestEvent` | `definitions["events.Event"]` |
| `IngestResult` | `definitions["response.Ingest"]` |
| `EventType` | `definitions["events.Type"]` |

**Frontend-only types** (maintained manually):

| Type | Purpose |
| --- | --- |
| `ServiceStatus` | `"checking" \| "ok" \| "unreachable"` health probe result. |
| `IngestProvider` | `"token" \| "codex"` connector authentication mode. |
| `ConnectorKind` | `"github" \| "slack" \| "jira" \| "filesystem" \| "googledrive" \| "notion" \| "sharepoint"` known connectors. |
| `CodexConnectorKind` | Subset of `ConnectorKind` that supports Codex SSE streaming. |
| `DirectSourceConnectorKind` | Complement of `CodexConnectorKind`; connectors that only use direct POST. |
| `CodexPlugin` | Name, `installed`, and `enabled` flags from the Codex status API. |
| `IngestRequest` | Unified request envelope sent to `postIngest`; collapses per-connector swagger types. |
| `ApiErrorBody` | Shape of `{ error?, message?, examples? }` returned on non-2xx responses, including source-scope examples for broad connector failures. |
| `SourceMetadataField` | Descriptor for a metadata field rendered inside `SourceConnector`. |
| `FindingsResult` | Aggregated findings response with mismatch, event, entity, and relationship counts for the insight panel. |
| `GraphData` | Graph response contract with flattened entities, relationships, summary stats, filtered/total counts, and display metadata used by the graph view model. |
| `SupportedFormat` | One row in a supported-formats table. |
| `SourceConnectorConfig` | Full static config that drives a `SourceConnector` card. |
| `WorkspaceList` | API response for registered local workspaces. |
| `WorkspaceSyncState` | Connector sync or connected-source registry row; `status: "connected"` means setup saved a live external source without ingesting it locally. |
| `ArtifactList` | API response for local ingested source artifacts. |
| `ChatQueryRequest` | Request body for source chat queries. |
| `ChatQueryResult` | Chat answer with intent, provider, answer text, artifact evidence, range, sync state, live evidence save status, and graph update status. |
| `ChatStreamState` | Frontend-only live query stream transcript with latest line, status, optional summary, and collapsed/expanded preference. |
| `GraphRelationship` | Persisted graph edge exposed by `/graph`, including source and evidence identifiers. |

---

## generated/

Contains `api.d.ts`, which is auto-generated from `apps/api/docs/swagger.json`. Do **not** edit it by hand. See [Type generation](../../README.md#type-generation).
