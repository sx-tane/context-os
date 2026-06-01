import type {
  ApiErrorBody,
  CodexConnectorKind,
  ConnectorKind,
  FindingsRequest,
  FindingsResult,
  GraphData,
  IngestRequest,
  IngestResult,
  ServiceStatus,
  WorkspaceRecord,
  WorkspaceStatus,
} from "$lib/types";

export const API_URL = "/api";

interface StreamHandlers {
  onLog?: (line: string) => void;
  onStatus?: (elapsed: number) => void;
  onResult?: (result: IngestResult) => void;
  onError?: (message: string) => void;
}

interface RequestOptions {
  signal?: AbortSignal;
}

interface SSEMessage {
  event: string;
  data: string;
}

export async function probeService(url: string): Promise<ServiceStatus> {
  try {
    const res = await fetch(`${url}/health`, {
      signal: AbortSignal.timeout(3000),
    });
    return res.ok ? "ok" : "unreachable";
  } catch {
    return "unreachable";
  }
}

export async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${API_URL}${path}`);
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
  const res = await fetch(`${API_URL}/${connector}/ingest`, {
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
  const res = await fetch(`${API_URL}/presentation/findings`, {
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
      body: responseBody as unknown as FindingsResult,
    };
  }
  return {
    ok: false,
    status: res.status,
    body: responseBody,
  };
}

export async function postFilesystemUpload(
  body: FormData,
  options: RequestOptions = {},
): Promise<
  | { ok: true; status: number; body: IngestResult }
  | { ok: false; status: number; body: ApiErrorBody }
> {
  const res = await fetch(`${API_URL}/filesystem/upload`, {
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
  body: { uri: string; token?: string; provider: "codex" },
  handlers: StreamHandlers,
  options: RequestOptions = {},
): Promise<void> {
  const res = await fetch(`${API_URL}/${connector}/ingest/stream`, {
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
      const parsed = parseJSON<{ message?: string }>(message.data);
      handlers.onError?.(parsed?.message ?? message.data);
    }
  });
}

export async function streamCodexReauth(
  plugin: string,
  onLog: (line: string) => void,
  options: RequestOptions = {},
): Promise<void> {
  const res = await fetch(
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
  const res = await fetch(`${API_URL}/codex/login`, {
    method: "POST",
    signal: options.signal,
  });
  await assertStreamResponse(res);
  await readLogStream(res, onLog);
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

/** Register or update a workspace by path. Returns the stored record or null on error. */
export async function upsertWorkspace(
  path: string,
  name: string,
): Promise<WorkspaceRecord | null> {
  try {
    const res = await fetch(`${API_URL}/workspace/upsert`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, name }),
    });
    if (!res.ok) return null;
    const body = await readJSON(res);
    return (body as unknown as WorkspaceRecord) ?? null;
  } catch {
    return null;
  }
}

/** Fetch workspace status (event counts, sync states). Returns null on error. */
export async function getWorkspaceStatus(
  path: string,
): Promise<WorkspaceStatus | null> {
  try {
    const res = await fetch(
      `${API_URL}/workspace/status?path=${encodeURIComponent(path)}`,
    );
    if (!res.ok) return null;
    return (await readJSON(res)) as unknown as WorkspaceStatus;
  } catch {
    return null;
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
    const res = await fetch(`${API_URL}/graph?${params.toString()}`);
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
  }

  const tail = parseSSEBlock(buffer);
  if (tail) onMessage(tail);
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
