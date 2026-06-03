# lib

Shared TypeScript modules for the ContextOS frontend. Everything in this directory is imported by routes and components using the `$lib` alias.

## Files

| File                                                     | Purpose                                                                                                                            |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| [`api.ts`](api.ts)                                       | HTTP and SSE client for the Go API. All network calls go through here.                                                             |
| [`types.ts`](types.ts)                                   | Canonical frontend type definitions. Split between re-exports of auto-generated API types and hand-maintained frontend-only types. |
| [`projectStore.ts`](projectStore.ts)                     | Svelte stores for workspace project state, chat history, selected connectors, protected demo/default workspaces, and backend workspace registration. |
| [`ingestRunner.ts`](ingestRunner.ts)                     | Orchestrates a single connector ingest run, branching between the direct `POST /ingest` path and the Codex SSE streaming path.     |
| [`reauthRunner.ts`](reauthRunner.ts)                     | Runs a Codex plugin re-auth SSE stream and refreshes Codex status when it finishes.                                                |
| [`sourceConnectorConfigs.ts`](sourceConnectorConfigs.ts) | Static configuration objects that drive the `SourceConnector` UI for each non-Codex connector (filesystem).                        |

---

## api.ts

Single source of truth for all communication with the Go API (`/api`).

**Exports**

| Symbol                                               | What it does                                                                                                                                     |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------- |
| `API_URL`                                            | Base path constant (`/api`).                                                                                                                     |
| `probeService(url)`                                  | `GET /health` with a 3 s timeout; returns `"ok"` or `"unreachable"`.                                                                             |
| `getJSON<T>(path)`                                   | Generic `GET` that deserialises JSON or returns `null` on any error.                                                                             |
| `postIngest(connector, body, opts)`                  | `POST /<connector>/ingest` with a JSON body; returns a typed discriminated union `{ ok: true, body: IngestResult }                               | { ok: false, body: ApiErrorBody }`. |
| `postFindings(body, opts)`                           | `POST /presentation/findings`; preserves backend JSON errors and converts network fetch failures into `{ ok: false, status: 0, body: { error: "api_unreachable", message } }`. |
| `postFilesystemUpload(formData, opts)`               | `POST /filesystem/upload` with `multipart/form-data`; same discriminated-union return.                                                           |
| `getWorkspaces()`                                    | Fetches registered API workspaces and returns an empty list when unavailable.                                                                     |
| `upsertWorkspace(path, name)`                        | Registers or updates a local workspace path.                                                                                                     |
| `deleteWorkspace(path)`                              | Calls `DELETE /workspace?path=...` and returns structured `{ ok, status, message? }` details so the route can remove local state while reporting backend/API failures. |
| `getWorkspaceStatus(path)`                           | Fetches workspace event/entity/mismatch counts and connector sync state.                                                                          |
| `getArtifacts(params)`                               | Queries local source artifacts from `GET /artifacts` by workspace, connector, source URI, date range, text, and limit.                            |
| `postChatQuery(body, opts)`                          | Sends chat questions to `POST /chat/query`; source questions use local artifacts first and may return Codex-backed live answers for configured sources. Network failures return a structured `api_unreachable` error instead of throwing raw `Failed to fetch`. |
| `streamCodexIngest(connector, body, handlers, opts)` | Opens an SSE stream to `POST /<connector>/ingest` for Codex-backed connectors; dispatches `onLog`, `onStatus`, `onResult`, and `onError` events. |
| `streamCodexReauth(plugin, onLog, opts)`             | Opens an SSE stream to `POST /codex/plugin-reauth?plugin=…`; forwards each log line to `onLog`.                                                  |
| `streamCodexLogin(onLog, opts)`                      | Opens an SSE stream to `POST /codex/login`; forwards each log line to `onLog`.                                                                   |

All streaming functions use a shared `readEventStream` helper that parses SSE blocks (`event: …\ndata: …\n\n`) from the response body reader.

---

## types.ts

Central type registry for the frontend.

**Auto-generated from swagger** (do not edit these directly — run `bun run codegen` to refresh):

| Type alias     | Generated source                 |
| -------------- | -------------------------------- |
| `IngestEvent`  | `definitions["events.Event"]`    |
| `IngestResult` | `definitions["response.Ingest"]` |
| `EventType`    | `definitions["events.Type"]`     |

**Frontend-only types** (maintained manually):

| Type                        | Purpose                                                                               |
| --------------------------- | ------------------------------------------------------------------------------------- |
| `ServiceStatus`             | `"checking" \| "ok" \| "unreachable"` — health probe result.                          |
| `IngestProvider`            | `"token" \| "codex"` — how a connector is authenticated.                              |
| `ConnectorKind`             | `"github" \| "slack" \| "jira" \| "filesystem" \| "googledrive" \| "notion" \| "sharepoint"` — all known connectors. |
| `CodexConnectorKind`        | Subset of `ConnectorKind` that supports Codex SSE streaming.                          |
| `DirectSourceConnectorKind` | Complement of `CodexConnectorKind`; connectors that only use direct POST.             |
| `CodexPlugin`               | Name, `installed`, and `enabled` flags from the Codex status API.                     |
| `IngestRequest`             | Unified request envelope sent to `postIngest`; collapses per-connector swagger types. |
| `ApiErrorBody`              | Shape of `{ error?, message? }` returned on non-2xx responses.                        |
| `SourceMetadataField`       | Descriptor for a metadata field rendered inside `SourceConnector`.                    |
| `SupportedFormat`           | One row in a supported-formats table (format name, extensions, extraction note).      |
| `SourceConnectorConfig`     | Full static config that drives a `SourceConnector` card.                              |
| `WorkspaceList`             | API response for registered local workspaces.                                        |
| `ArtifactList`              | API response for local ingested source artifacts.                                    |
| `ChatQueryRequest`          | Request body for source chat queries.                                                |
| `ChatQueryResult`           | Chat answer with intent, provider, answer text, artifact evidence, range, and sync state. |
| `GraphRelationship`         | Persisted graph edge exposed by `/graph`, including source and evidence identifiers. |
| `GraphData`                 | Graph response with flattened entities, relationships, and summary stats.            |

---

## findingsAggregator.ts

Merges per-source `postFindings` responses into one `FindingsResult` for the homepage. It combines mismatch arrays, sums mismatch/event/entity counts, and builds the chat summary text that distinguishes a successful zero-finding analysis from source failures.

---

## projectStore.ts

Maintains local workspace state in browser storage and registers user-created workspaces with the backend. The store always exposes the default workspace and a protected `contextos-demo` workspace; neither protected workspace is marked removed or deleted from local storage. The demo workspace is local-only and is not registered with the backend when opened.

---

## ingestRunner.ts

Coordinates a complete ingest lifecycle for a single connector so components stay thin.

**Exported function**

```ts
runConnectorIngest(options: IngestRunnerOptions): Promise<void>
```

`IngestRunnerOptions` carries:

| Field                                                             | Purpose                                                                                                                         |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `connector`                                                       | Which `ConnectorKind` to call.                                                                                                  |
| `workspace_id`                                                    | Optional workspace path or ID forwarded to direct and Codex ingest requests so backend persistence is workspace-scoped.         |
| `uri`, `token`, `content`, `cursor`, `metadata`                   | Ingest payload fields.                                                                                                          |
| `provider`                                                        | `"codex"` routes through `streamCodexIngest`; anything else uses `postIngest`.                                                  |
| `setLoading`, `setError`, `setResult`, `setLiveLog`, `setElapsed` | Svelte reactive setters the runner calls as state changes.                                                                      |
| `isCurrent()`                                                     | Optional guard: if it returns `false`, all setter calls are silently dropped (prevents stale updates after a component switch). |
| `signal`                                                          | `AbortSignal` for cancellation; abort errors are silently swallowed.                                                            |

**Flow**

```
provider === "codex"  →  streamCodexIngest  →  onLog / onStatus / onResult / onError
provider !== "codex"  →  postIngest  →  result or error body
```

A `setInterval` timer increments `elapsed` every second during Codex streaming and is cleared in `finally`.

Connector components should pass the current `$project.workspacePath` as `workspace_id` so all ingest providers write to the active workspace.

---

## reauthRunner.ts

Wrapper around `streamCodexReauth` for the Codex plugin re-auth flow.

> **Not currently wired into the UI.** To reconnect a plugin to a different account, run
> `codex plugin remove <plugin>@openai-curated && codex plugin add <plugin>@openai-curated`
> directly in your terminal. Tracked in issue: **Add frontend Codex plugin re-auth flow**.

**Exported function**

```ts
runCodexReauth(options: ReauthRunnerOptions): Promise<void>
```

`ReauthRunnerOptions` carries:

| Field                               | Purpose                                                           |
| ----------------------------------- | ----------------------------------------------------------------- |
| `plugin`                            | Plugin name passed to the re-auth endpoint.                       |
| `refreshCodexStatus()`              | Called in `finally` to reload Codex status after the stream ends. |
| `setPlugin`, `setLog`, `setRunning` | Svelte reactive setters.                                          |
| `isCurrent()`                       | Same stale-update guard as `IngestRunnerOptions`.                 |
| `signal`                            | `AbortSignal`; abort errors are silently swallowed.               |

---

## sourceConnectorConfigs.ts

Static array of `SourceConnectorConfig` objects. One entry per `DirectSourceConnectorKind`.

Each config is consumed by the route page to render a `SourceConnector` card without hardcoding connector details in the UI. Currently contains a single entry for the filesystem connector. Add a new object here when a new direct (non-Codex) connector is introduced.

---

## generated/

Contains `api.d.ts`, which is auto-generated from `apps/api/_docs/swagger.json`. Do **not** edit it by hand. See [Type generation](../../../README.md#type-generation).
