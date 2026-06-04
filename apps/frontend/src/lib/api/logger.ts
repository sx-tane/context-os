export interface APIRequestDescription {
  id: string;
  method: string;
  url: string;
  body: string;
}

declare const __CONTEXTOS_DEBUG_LOGS__: string | undefined;

let requestSequence = 0;
let helperInstalled = false;

export function prepareAPIRequest(
  input: RequestInfo | URL,
  init: RequestInit = {},
): { description: APIRequestDescription; init: RequestInit } {
  const requestID = nextRequestID();
  const description = describeRequest(input, init, requestID);
  return {
    description,
    init: {
      ...init,
      headers: withRequestID(init.headers, requestID),
    },
  };
}

export function logAPIRequestStart(request: APIRequestDescription): void {
  debugAPI(
    `-> ${request.method} ${request.url} id=${request.id}${request.body ? ` body=${request.body}` : ""}`,
  );
}

export function logAPIRequestDone(
  request: APIRequestDescription,
  status: number,
  durationMS: number,
): void {
  debugAPI(
    `<- ${status} ${request.method} ${request.url} id=${request.id} ${durationMS}ms`,
  );
}

export function logAPIRequestError(
  request: APIRequestDescription,
  durationMS: number,
  error: unknown,
): void {
  debugAPI(
    `xx ${request.method} ${request.url} id=${request.id} ${durationMS}ms ${error instanceof Error ? error.message : String(error)}`,
  );
}

export function frontendDebugEnabled(): boolean {
  installAPITraceHelper();
  if (viteDebugEnabled()) return true;
  const storage = globalThisLocalStorage();
  return storage?.getItem("contextos_debug_api") === "1";
}

function debugAPI(message: string): void {
  if (!frontendDebugEnabled()) return;
  console.debug(`[api] ${message}`);
}

function nextRequestID(): string {
  requestSequence += 1;
  return `web-${Date.now().toString(36)}-${requestSequence.toString(36)}`;
}

function describeRequest(
  input: RequestInfo | URL,
  init: RequestInit,
  requestID: string,
): APIRequestDescription {
  const method =
    init.method ?? (input instanceof Request ? input.method : "GET");
  return {
    id: requestID,
    method: method.toUpperCase(),
    url: String(input),
    body: summarizeBody(init.body),
  };
}

function summarizeBody(body: BodyInit | null | undefined): string {
  if (!body) return "";
  if (typeof body === "string") {
    return body.length > 220 ? `${body.slice(0, 220)}...` : body;
  }
  if (body instanceof FormData) return "[FormData]";
  return `[${body.constructor.name}]`;
}

function withRequestID(
  headers: HeadersInit | undefined,
  requestID: string,
): HeadersInit {
  if (headers instanceof Headers || Array.isArray(headers)) {
    const next = new Headers(headers);
    if (!next.has("X-ContextOS-Request-ID")) {
      next.set("X-ContextOS-Request-ID", requestID);
    }
    return next;
  }
  const next = { ...(headers ?? {}) };
  if (!("X-ContextOS-Request-ID" in next)) {
    next["X-ContextOS-Request-ID"] = requestID;
  }
  return next;
}

function viteDebugEnabled(): boolean {
  return (
    typeof __CONTEXTOS_DEBUG_LOGS__ !== "undefined" &&
    __CONTEXTOS_DEBUG_LOGS__ === "1"
  );
}

function globalThisLocalStorage(): Storage | null {
  const maybeStorage = (globalThis as { localStorage?: Storage }).localStorage;
  return maybeStorage ?? null;
}

function installAPITraceHelper(): void {
  if (helperInstalled || typeof window === "undefined") return;
  helperInstalled = true;
  const target = window as Window & {
    contextosAPITrace?: (enabled?: boolean) => void;
  };
  if (target.contextosAPITrace) return;
  target.contextosAPITrace = (enabled = true) => {
    const storage = globalThisLocalStorage();
    if (!storage) return;
    if (enabled) {
      storage.setItem("contextos_debug_api", "1");
      console.info(
        "[api] browser request tracing enabled. Reload or keep using the app; logs include method, /api path, body preview, status, duration, and X-ContextOS-Request-ID.",
      );
      return;
    }
    storage.removeItem("contextos_debug_api");
    console.info("[api] browser request tracing disabled.");
  };
}
