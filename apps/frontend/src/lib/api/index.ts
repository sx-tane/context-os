import type {
  ApiErrorBody,
  ArtifactList,
  ActivityCleanupResult,
  ChatQueryResult,
  CodexConnectorKind,
  CodexSourceList,
  ConnectorKind,
  FindingsRequest,
  FindingsResult,
  GraphCleanupResult,
  GraphData,
  IngestRequest,
  IngestResult,
  ServiceStatus,
  SourceRegistrationRequest,
  WorkspaceRecord,
  WorkspaceSyncState,
  WorkspaceStatus,
} from "$lib/types";
import type {
  AnalysisBasketPayload,
  ChatQueryRequest,
  ChatSessionResetRequest,
  FindingActionsPayload,
} from "$lib/api/types";
import {
  logAPIRequestDone,
  logAPIRequestError,
  logAPIRequestStart,
  prepareAPIRequest,
} from "$lib/api/logger";

export const API_URL = "/api";

interface StreamHandlers {
  onLog?: (line: string) => void;
  onStatus?: (elapsed: number) => void;
  onResult?: (result: IngestResult) => void;
  onError?: (message: string) => void;
}

interface ChatStreamHandlers {
  onLog?: (line: string) => void;
  onStatus?: (elapsed: number) => void;
  onAnswer?: (result: ChatQueryResult) => void;
  onResult?: (result: ChatQueryResult) => void;
  onError?: (message: string) => void;
}

interface RequestOptions {
  signal?: AbortSignal;
}

export interface DeleteWorkspaceResult {
  ok: boolean;
  status: number;
  message?: string;
}

interface SSEMessage {
  event: string;
  data: string;
}

export async function apiFetch(
  input: RequestInfo | URL,
  init: RequestInit = {},
): Promise<Response> {
  const started = performance.now();
  const request = prepareAPIRequest(input, init);
  logAPIRequestStart(request.description);
  try {
    const res = await fetch(input, request.init);
    logAPIRequestDone(
      request.description,
      res.status,
      Math.round(performance.now() - started),
    );
    return res;
  } catch (error) {
    logAPIRequestError(
      request.description,
      Math.round(performance.now() - started),
      error,
    );
    throw error;
  }
}

export async function probeService(url: string): Promise<ServiceStatus> {
  try {
    const controller = new AbortController();
    const res = await withTimeout(apiFetch(`${url}/health`, {
      signal: controller.signal,
    }), 3000, () => {
      controller.abort();
    });
    return res.ok ? "ok" : "unreachable";
  } catch {
    return "unreachable";
  }
}

async function withTimeout<T>(
  promise: Promise<T>,
  milliseconds: number,
  onTimeout?: () => void,
): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => {
      onTimeout?.();
      reject(new Error(`request timed out after ${milliseconds}ms`));
    }, milliseconds);
  });
  try {
    return await Promise.race([promise, timeout]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

export async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const res = await apiFetch(`${API_URL}${path}`);
    if (!res.ok) return null;
    return (await readJSON(res)) as T;
  } catch {
    return null;
  }
}

export async function postIngest(
  connector: ConnectorKind,
  body: IngestRequest,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: IngestResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  const res = await apiFetch(`${API_URL}/${connector}/ingest`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: options.signal,
  });
  const responseBody = await readJSON(res);
  if (res.ok) {
    return {
      ok: true,
      status: res.status,
      body: responseBody as unknown as IngestResult,
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

export async function postFindings(
  body: FindingsRequest,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: FindingsResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  let res: Response;
  try {
    res = await apiFetch(`${API_URL}/presentation/findings`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: options.signal,
    });
  } catch (error) {
    if (isAbortError(error)) throw error;
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message:
          "API is unreachable. Start the API with scripts/start-all.sh or check the frontend /api proxy.",
      },
    };
  }
  const responseBody = await readJSON(res);
  if (res.ok) {
    return {
      ok: true,
      status: res.status,
      body: responseBody as unknown as FindingsResult,
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

function isAbortError(error: unknown) {
  return (
    error instanceof DOMException && error.name === "AbortError"
  );
}

export async function postFilesystemUpload(
  body: FormData,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: IngestResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  const res = await apiFetch(`${API_URL}/filesystem/upload`, {
    method: "POST",
    body,
    signal: options.signal,
  });
  const responseBody = await readJSON(res);
  if (res.ok) {
    return {
      ok: true,
      status: res.status,
      body: responseBody as unknown as IngestResult,
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

export async function streamCodexIngest(
  connector: CodexConnectorKind,
  body: { workspace_id?: string; uri: string; token?: string; provider: "codex" },
  handlers: StreamHandlers,
  options: RequestOptions = {},
): Promise<void> {
  const res = await apiFetch(`${API_URL}/${connector}/ingest/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: options.signal,
  });
  await assertStreamResponse(res);
  await readEventStream(res, (message) => {
    if (message.event === "log") {
      handlers.onLog?.(message.data);
    } else if (message.event === "status") {
      const status = parseJSON<{ elapsed?: number }>(message.data);
      if (status?.elapsed !== undefined) handlers.onStatus?.(status.elapsed);
    } else if (message.event === "result") {
      const result = parseJSON<IngestResult>(message.data);
      if (result) handlers.onResult?.(result);
    } else if (message.event === "error") {
      const parsed = parseJSON<{ error?: string; message?: string }>(message.data);
      if (options.signal?.aborted || parsed?.error === "query_canceled") {
        throw new DOMException(parsed?.message ?? "aborted", "AbortError");
      }
      handlers.onError?.(parsed?.message ?? message.data);
    }
  });
}

export async function streamCodexReauth(
  plugin: string,
  onLog: (line: string) => void,
  options: RequestOptions = {},
): Promise<void> {
  const res = await apiFetch(
    `${API_URL}/codex/plugin-reauth?plugin=${encodeURIComponent(plugin)}`,
    {
      method: "POST",
      signal: options.signal,
    },
  );
  await assertStreamResponse(res);
  await readLogStream(res, onLog);
}

export async function streamCodexLogin(
  onLog: (line: string) => void,
  options: RequestOptions = {},
): Promise<void> {
  const res = await apiFetch(`${API_URL}/codex/login`, {
    method: "POST",
    signal: options.signal,
  });
  await assertStreamResponse(res);
  await readLogStream(res, onLog);
}

export async function getCodexSources(
  connector: CodexConnectorKind,
): Promise<CodexSourceList | null> {
  try {
    const res = await apiFetch(
      `${API_URL}/codex/sources?connector=${encodeURIComponent(connector)}`,
    );
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as CodexSourceList;
  } catch {
    return null;
  }
}

async function readLogStream(
  res: Response,
  onLog: (line: string) => void,
): Promise<void> {
  await readEventStream(res, (message) => {
    if (message.data) onLog(message.data);
  });
}

// ---- Workspace API helpers ----

/** Fetch all registered workspaces. Returns an empty list on error. */
export async function getWorkspaces(): Promise<WorkspaceRecord[]> {
  try {
    const res = await apiFetch(`${API_URL}/workspace`);
    if (!res.ok) return [];
    return normalizeWorkspaceList(await readJSON(res));
  } catch {
    return [];
  }
}

/** Register or update a workspace by path. Returns the stored record or null on error. */
export async function upsertWorkspace(
  path: string,
  name: string,
): Promise<WorkspaceRecord | null> {
  try {
    const res = await apiFetch(`${API_URL}/workspace/upsert`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, name }),
    });
    if (!res.ok) return null;
    const body = await readJSON(res);
    return normalizeWorkspaceRecord(body);
  } catch {
    return null;
  }
}

export async function resetWorkspace(
  path: string,
  name?: string,
): Promise<WorkspaceStatus | null> {
  const body: { path: string; name?: string } = { path };
  const trimmedName = name?.trim();
  if (trimmedName) body.name = trimmedName;

  try {
    const res = await apiFetch(`${API_URL}/workspace/reset`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) return null;
    return normalizeWorkspaceStatus(await readJSON(res));
  } catch {
    return null;
  }
}

export async function probeWorkspaceService(
  workspaceID: string,
): Promise<ServiceStatus> {
  try {
    const res = await apiFetch(
      `${API_URL}/workspace/status?path=${encodeURIComponent(workspaceID)}`,
    );
    const contentType = res.headers?.get("content-type") ?? "";
    if (res.ok || (res.status === 404 && contentType.includes("application/json"))) {
      return "ok";
    }
    return "unreachable";
  } catch {
    return "unreachable";
  }
}

export async function postWorkspaceSource(
  body: SourceRegistrationRequest,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: WorkspaceSyncState }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  let res: Response;
  try {
    res = await apiFetch(`${API_URL}/workspace/source`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: options.signal,
    });
  } catch (error) {
    if (isAbortError(error)) throw error;
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message:
          "API is unreachable. Start the API with scripts/start-all.sh or check the frontend /api proxy.",
      },
    };
  }

  const responseBody = await readJSON(res);
  if (res.ok) {
    const sync = normalizeWorkspaceSync(responseBody);
    return {
      ok: true,
      status: res.status,
      body: sync ?? (responseBody as unknown as WorkspaceSyncState),
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

export async function deleteWorkspace(path: string): Promise<DeleteWorkspaceResult> {
  try {
    const res = await apiFetch(
      `${API_URL}/workspace?path=${encodeURIComponent(path)}`,
      { method: "DELETE" },
    );
    if (res.ok) return { ok: true, status: res.status };
    const body = await readJSON(res);
    return {
      ok: false,
      status: res.status,
      message: body.message ?? body.error ?? `Request failed with status ${res.status}`,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      message: "API is unreachable. Nothing was removed.",
    };
  }
}

/** Fetch workspace status (event counts, sync states). Returns null on error. */
export async function getWorkspaceStatus(
  path: string,
): Promise<WorkspaceStatus | null> {
  try {
    const res = await apiFetch(
      `${API_URL}/workspace/status?path=${encodeURIComponent(path)}`,
    );
    if (!res.ok) return null;
    return normalizeWorkspaceStatus(await readJSON(res));
  } catch {
    return null;
  }
}

function normalizeWorkspaceList(body: unknown): WorkspaceRecord[] {
  const record = asRecord(body);
  const rawWorkspaces = asArray(record?.workspaces ?? record?.Workspaces);
  return rawWorkspaces
    .map(normalizeWorkspaceRecord)
    .filter((workspace): workspace is WorkspaceRecord => workspace !== null);
}

function normalizeWorkspaceRecord(body: unknown): WorkspaceRecord | null {
  const record = asRecord(body);
  if (!record) return null;
  const path = stringField(record, "path", "Path").trim();
  if (!path) return null;
  const name =
    stringField(record, "name", "Name").trim() ||
    path.split("/").filter(Boolean).pop() ||
    "workspace";
  const workspace: WorkspaceRecord = {
    id: stringField(record, "id", "ID").trim() || path,
    name,
    path,
  };
  const createdAt = stringField(record, "created_at", "CreatedAt").trim();
  if (createdAt) workspace.created_at = createdAt;
  const updatedAt = stringField(record, "updated_at", "UpdatedAt").trim();
  if (updatedAt) workspace.updated_at = updatedAt;
  return workspace;
}

function normalizeWorkspaceStatus(body: unknown): WorkspaceStatus {
  const record = asRecord(body);
  if (!record) return {};
  const status: WorkspaceStatus = {};
  const workspace = normalizeWorkspaceRecord(record.workspace ?? record.Workspace);
  if (workspace) status.workspace = workspace;
  const workspaceCount = numberField(record, "workspace_count", "WorkspaceCount");
  if (workspaceCount !== undefined) status.workspace_count = workspaceCount;
  const eventCount = numberField(record, "event_count", "EventCount");
  if (eventCount !== undefined) status.event_count = eventCount;
  const entityCount = numberField(record, "entity_count", "EntityCount");
  if (entityCount !== undefined) status.entity_count = entityCount;
  const relationshipCount = numberField(record, "relationship_count", "RelationshipCount");
  if (relationshipCount !== undefined) status.relationship_count = relationshipCount;
  const mismatchCount = numberField(record, "mismatch_count", "MismatchCount");
  if (mismatchCount !== undefined) status.mismatch_count = mismatchCount;
  const connectorSyncCount = numberField(record, "connector_sync_count", "ConnectorSyncCount");
  if (connectorSyncCount !== undefined) status.connector_sync_count = connectorSyncCount;
  const auditCount = numberField(record, "audit_count", "AuditCount");
  if (auditCount !== undefined) status.audit_count = auditCount;
  const syncs = asArray(record.syncs ?? record.Syncs)
    .map(normalizeWorkspaceSync)
    .filter((sync): sync is WorkspaceSyncState => sync !== null);
  if (syncs.length > 0) status.syncs = syncs;
  return status;
}

function normalizeWorkspaceSync(body: unknown): WorkspaceSyncState | null {
  const record = asRecord(body);
  if (!record) return null;
  const connector = stringField(record, "connector", "Connector").trim();
  const sourceURI = stringField(record, "source_uri", "SourceURI").trim();
  if (!connector && !sourceURI) return null;
  const sync: WorkspaceSyncState = {
    connector,
    source_uri: sourceURI,
  };
  const cursor = stringField(record, "cursor", "Cursor").trim();
  if (cursor) sync.cursor = cursor;
  const lastSyncedAt = stringField(record, "last_synced_at", "LastSyncedAt").trim();
  if (lastSyncedAt) sync.last_synced_at = lastSyncedAt;
  const eventCount = numberField(record, "event_count", "EventCount");
  if (eventCount !== undefined) sync.event_count = eventCount;
  const currentStatus = stringField(record, "status", "Status").trim();
  if (currentStatus) sync.status = currentStatus;
  const lastError = stringField(record, "last_error", "LastError").trim();
  if (lastError) sync.last_error = lastError;
  return sync;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function stringField(
  record: Record<string, unknown>,
  snakeKey: string,
  legacyKey: string,
): string {
  const value = record[snakeKey] ?? record[legacyKey];
  return typeof value === "string" ? value : "";
}

function numberField(
  record: Record<string, unknown>,
  snakeKey: string,
  legacyKey: string,
): number | undefined {
  const value = record[snakeKey] ?? record[legacyKey];
  return typeof value === "number" ? value : undefined;
}

/** Query local source artifacts from the workspace database. */
export async function getArtifacts(params: {
  workspace_id: string;
  connector?: string;
  source_uri?: string;
  q?: string;
  since?: string;
  until?: string;
  limit?: number;
}): Promise<ArtifactList | null> {
  try {
    const query = new URLSearchParams({ workspace_id: params.workspace_id });
    if (params.connector) query.set("connector", params.connector);
    if (params.source_uri) query.set("source_uri", params.source_uri);
    if (params.q) query.set("q", params.q);
    if (params.since) query.set("since", params.since);
    if (params.until) query.set("until", params.until);
    if (params.limit) query.set("limit", String(params.limit));
    const res = await apiFetch(`${API_URL}/artifacts?${query.toString()}`);
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as ArtifactList;
  } catch {
    return null;
  }
}

/** Delete explicitly selected old noisy live-chat evidence artifacts. */
export async function cleanupLiveEvidence(
  workspaceID: string,
): Promise<
  | { ok: true; status: number; body: ActivityCleanupResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  try {
    const res = await apiFetch(
      `${API_URL}/artifacts/live-evidence/cleanup?workspace_id=${encodeURIComponent(workspaceID)}`,
      { method: "POST" },
    );
    const responseBody = await readJSON(res);
    if (res.ok) {
      return {
        ok: true,
        status: res.status,
        body: responseBody as unknown as ActivityCleanupResult,
      };
    }
    return {
      ok: false,
      status: res.status,
      body: responseBody,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Activity cleanup did not run.",
      },
    };
  }
}

/** Delete user-selected Activity artifacts from the local workspace database. */
export async function deleteArtifacts(body: {
  workspace_id: string;
  ids: string[];
}): Promise<
  | { ok: true; status: number; body: ActivityCleanupResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  try {
    const res = await apiFetch(`${API_URL}/artifacts/delete`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const responseBody = await readJSON(res);
    if (res.ok) {
      return {
        ok: true,
        status: res.status,
        body: responseBody as unknown as ActivityCleanupResult,
      };
    }
    return {
      ok: false,
      status: res.status,
      body: responseBody,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Activity delete did not run.",
      },
    };
  }
}

export async function getAnalysisBasket(
  workspaceID: string,
): Promise<AnalysisBasketPayload | null> {
  try {
    const query = new URLSearchParams({ workspace_id: workspaceID });
    const res = await apiFetch(`${API_URL}/workspace/analysis-basket?${query.toString()}`);
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as AnalysisBasketPayload;
  } catch {
    return null;
  }
}

export async function putAnalysisBasket(
  body: AnalysisBasketPayload,
): Promise<
  | { ok: true; status: number; body: AnalysisBasketPayload }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  return putWorkspaceUIState("/workspace/analysis-basket", body);
}

export async function getFindingActions(
  workspaceID: string,
): Promise<FindingActionsPayload | null> {
  try {
    const query = new URLSearchParams({ workspace_id: workspaceID });
    const res = await apiFetch(`${API_URL}/workspace/finding-actions?${query.toString()}`);
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as FindingActionsPayload;
  } catch {
    return null;
  }
}

export async function putFindingActions(
  body: FindingActionsPayload,
): Promise<
  | { ok: true; status: number; body: FindingActionsPayload }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  return putWorkspaceUIState("/workspace/finding-actions", body);
}

async function putWorkspaceUIState<T>(
  path: string,
  body: T,
): Promise<
  | { ok: true; status: number; body: T }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  try {
    const res = await apiFetch(`${API_URL}${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const responseBody = await readJSON(res);
    if (res.ok) {
      return {
        ok: true,
        status: res.status,
        body: responseBody as unknown as T,
      };
    }
    return {
      ok: false,
      status: res.status,
      body: responseBody,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Workspace UI state did not save.",
      },
    };
  }
}

/** Permanently delete backend-classified low-signal graph rows for a workspace. */
export async function cleanupGraphNoise(
  workspaceID: string,
): Promise<
  | { ok: true; status: number; body: GraphCleanupResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  try {
    const res = await apiFetch(
      `${API_URL}/graph/cleanup?workspace_id=${encodeURIComponent(workspaceID)}`,
      { method: "POST" },
    );
    const responseBody = await readJSON(res);
    if (res.ok) {
      return {
        ok: true,
        status: res.status,
        body: responseBody as unknown as GraphCleanupResult,
      };
    }
    return {
      ok: false,
      status: res.status,
      body: responseBody,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Graph cleanup did not run.",
      },
    };
  }
}

/** Permanently delete one graph entity and relationships touching it. */
export async function deleteGraphEntity(
  workspaceID: string,
  entityID: string,
): Promise<
  | { ok: true; status: number; body: GraphCleanupResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  try {
    const res = await apiFetch(
      `${API_URL}/graph/entity?workspace_id=${encodeURIComponent(workspaceID)}&entity_id=${encodeURIComponent(entityID)}`,
      { method: "DELETE" },
    );
    const responseBody = await readJSON(res);
    if (res.ok) {
      return {
        ok: true,
        status: res.status,
        body: responseBody as unknown as GraphCleanupResult,
      };
    }
    return {
      ok: false,
      status: res.status,
      body: responseBody,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Graph entity delete did not run.",
      },
    };
  }
}

/** Ask a deterministic local chat question over workspace source artifacts. */
export async function postChatQuery(
  body: ChatQueryRequest,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: ChatQueryResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  let res: Response;
  try {
    res = await apiFetch(`${API_URL}/chat/query`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: options.signal,
    });
  } catch (error) {
    if (isAbortError(error)) throw error;
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message:
          "API is unreachable. Start the API with scripts/start-all.sh or check the frontend /api proxy.",
      },
    };
  }
  if (options.signal?.aborted) {
    throw new DOMException("aborted", "AbortError");
  }
  const responseBody = await readJSON(res);
  if (options.signal?.aborted) {
    throw new DOMException("aborted", "AbortError");
  }
  if (res.ok) {
    return {
      ok: true,
      status: res.status,
      body: responseBody as unknown as ChatQueryResult,
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

export async function streamChatQuery(
  body: ChatQueryRequest,
  handlers: ChatStreamHandlers,
  options: RequestOptions = {},
): Promise<void> {
  const res = await apiFetch(`${API_URL}/chat/query/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: options.signal,
  });
  await assertStreamResponse(res);
  await readEventStream(res, (message) => {
    if (message.event === "log") {
      handlers.onLog?.(message.data);
    } else if (message.event === "status") {
      const status = parseJSON<{ elapsed?: number }>(message.data);
      if (status?.elapsed !== undefined) handlers.onStatus?.(status.elapsed);
    } else if (message.event === "answer") {
      const result = parseJSON<ChatQueryResult>(message.data);
      if (result) handlers.onAnswer?.(result);
    } else if (message.event === "result") {
      const result = parseJSON<ChatQueryResult>(message.data);
      if (result) handlers.onResult?.(result);
    } else if (message.event === "error") {
      const parsed = parseJSON<{ error?: string; message?: string }>(message.data);
      if (options.signal?.aborted || parsed?.error === "query_canceled") {
        throw new DOMException(parsed?.message ?? "aborted", "AbortError");
      }
      handlers.onError?.(parsed?.message ?? message.data);
    }
  });
  if (options.signal?.aborted) {
    throw new DOMException("aborted", "AbortError");
  }
}

export async function resetChatSession(
  body: ChatSessionResetRequest,
  options: RequestOptions = {},
): Promise<{ ok: boolean; status: number; body?: ApiErrorBody }> {
  try {
    const res = await apiFetch(`${API_URL}/chat/session/reset`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: options.signal,
    });
    if (res.ok) {
      return { ok: true, status: res.status };
    }
    return { ok: false, status: res.status, body: await readJSON(res) };
  } catch (error) {
    if (isAbortError(error)) throw error;
    return {
      ok: false,
      status: 0,
      body: {
        error: "api_unreachable",
        message: "API is unreachable. Chat was cleared locally only.",
      },
    };
  }
}

/** Fetch entity graph data for a workspace, optionally filtered by entity type. */
export async function getGraphData(
  workspacePath: string,
  entityType?: string,
): Promise<GraphData | null> {
  try {
    const params = new URLSearchParams({ workspace_id: workspacePath });
    if (entityType) params.set("entity_type", entityType);
    const res = await apiFetch(`${API_URL}/graph?${params.toString()}`);
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as GraphData;
  } catch {
    return null;
  }
}

async function readEventStream(
  res: Response,
  onMessage: (message: SSEMessage) => void,
): Promise<void> {
  if (!res.body) throw new Error("No response body");

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const blocks = buffer.split("\n\n");
      buffer = blocks.pop() ?? "";
      for (const block of blocks) {
        const message = parseSSEBlock(block);
        if (message) onMessage(message);
      }
      const partial = parseSSEBlock(buffer);
      if (partial && buffer.endsWith("\n")) {
        onMessage(partial);
        buffer = "";
      }
    }

    const tail = parseSSEBlock(buffer);
    if (tail) onMessage(tail);
  } finally {
    reader.releaseLock();
  }
}

function parseSSEBlock(block: string): SSEMessage | null {
  const eventLine = block.split("\n").find((line) => line.startsWith("event:"));
  const dataLines = block
    .split("\n")
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trim());

  if (!eventLine || dataLines.length === 0) return null;
  return {
    event: eventLine.slice(6).trim(),
    data: dataLines.join("\n"),
  };
}

async function assertStreamResponse(res: Response): Promise<void> {
  if (res.ok) return;
  const body = await readJSON(res);
  const message =
    body.message ?? body.error ?? `Request failed with status ${res.status}`;
  throw new Error(message);
}

async function readJSON(
  res: Response,
): Promise<ApiErrorBody & Record<string, unknown>> {
  const text = await res.text();
  if (!text.trim()) return {};
  const parsed = parseJSON<ApiErrorBody & Record<string, unknown>>(text);
  if (parsed) return parsed;
  return { message: text };
}

function parseJSON<T>(text: string): T | null {
  try {
    return JSON.parse(text) as T;
  } catch {
    return null;
  }
}
