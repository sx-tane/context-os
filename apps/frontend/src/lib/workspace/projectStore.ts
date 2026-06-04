import { writable, get } from "svelte/store";
import type {
  ChatMessage,
  ConnectorKind,
  ConnectorKnowledge,
  KnowledgeStatus,
  ProjectState,
  WorkspaceRecord,
  WorkspaceStatus,
} from "$lib/types";
import {
  getWorkspaces,
  resetChatSession,
  upsertWorkspace,
  getWorkspaceStatus,
} from "$lib/api";
import { applyWorkspaceSyncsToConnectors } from "$lib/workspace/statusMapping";

const STORAGE_KEY_PREFIX = "contextos_project_";
const CHAT_KEY_PREFIX = "contextos_chat_";
const WORKSPACE_LIST_KEY = "contextos_workspaces";
const ACTIVE_WORKSPACE_KEY = "contextos_workspace_path";
const MAX_CHAT_MESSAGES = 200;
const MAX_STREAM_LINES = 80;
const MAX_MESSAGE_TEXT_LENGTH = 30000;
const MAX_STREAM_TEXT_LENGTH = 4000;
const MAX_CARD_ARTIFACTS = 20;
const MAX_ARTIFACT_TEXT_LENGTH = 5000;
const MAX_WORKSPACES = 100;
const MAX_CONNECTORS = 100;

// Default workspace key — replaced by user selection or URL param.
export const DEFAULT_WORKSPACE_PATH =
  import.meta.env.VITE_CONTEXTOS_DEFAULT_WORKSPACE?.trim() ||
  "contextos-default";
export const DEMO_WORKSPACE_PATH = "contextos-demo";

function getLocalStorage(): Storage | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage;
}

function loadActiveWorkspacePath(): string {
  return getLocalStorage()?.getItem(ACTIVE_WORKSPACE_KEY) || DEFAULT_WORKSPACE_PATH;
}

function cleanWorkspacePath(path?: string): string {
  const cleanPath = path?.trim();
  return cleanPath || DEFAULT_WORKSPACE_PATH;
}

function trimText(text: string | undefined, maxLength: number): string {
  if (!text) return "";
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength)}\n\n[trimmed ${text.length - maxLength} characters from cached local state]`;
}

function trimTextList(values: string[] | undefined): string[] | undefined {
  return Array.isArray(values)
    ? values.slice(0, 12).map((value) => trimText(value, MAX_STREAM_TEXT_LENGTH))
    : undefined;
}

function normalizeProject(raw: ProjectState, fallbackPath: string): ProjectState {
  const workspacePath = cleanWorkspacePath(raw.workspacePath || fallbackPath);
  return {
    ...raw,
    workspacePath,
    name: raw.name || workspacePath.split("/").filter(Boolean).pop() || "project",
    createdAt: raw.createdAt || new Date().toISOString(),
    connectors: Array.isArray(raw.connectors)
      ? raw.connectors.slice(0, MAX_CONNECTORS)
      : [],
  };
}

function loadProject(path: string): ProjectState {
  const cleanPath = cleanWorkspacePath(path);
  if (cleanPath === DEMO_WORKSPACE_PATH) return demoProject();
  try {
    const storage = getLocalStorage();
    if (!storage) return defaultProject(cleanPath);
    const raw = storage.getItem(STORAGE_KEY_PREFIX + cleanPath);
    if (raw) {
      return normalizeProject(
        JSON.parse(raw) as ProjectState,
        cleanPath,
      );
    }
  } catch {
    // ignore parse errors
  }
  return defaultProject(cleanPath);
}

function defaultProject(path: string): ProjectState {
  const cleanPath = cleanWorkspacePath(path);
  return {
    workspacePath: cleanPath,
    name: cleanPath.split("/").filter(Boolean).pop() ?? "project",
    createdAt: new Date().toISOString(),
    connectors: [],
  };
}

function demoProject(): ProjectState {
  return {
    workspacePath: DEMO_WORKSPACE_PATH,
    name: "Demo Workspace",
    createdAt: "2026-01-01T00:00:00.000Z",
    knowledgeInstalledAt: "2026-01-01T00:00:00.000Z",
    connectors: [
      {
        connector: "github",
        uri: "context-os/demo-api",
        status: "ready",
        lastIngestedAt: "2026-01-01T09:15:00.000Z",
        eventCount: 18,
      },
      {
        connector: "slack",
        uri: "#launch-review",
        status: "ready",
        lastIngestedAt: "2026-01-01T09:20:00.000Z",
        eventCount: 24,
      },
      {
        connector: "jira",
        uri: "DEMO",
        status: "ready",
        lastIngestedAt: "2026-01-01T09:25:00.000Z",
        eventCount: 12,
      },
    ],
  };
}

function saveProject(state: ProjectState): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    const safeState = normalizeProject(state, state.workspacePath);
    storage.setItem(
      STORAGE_KEY_PREFIX + safeState.workspacePath,
      JSON.stringify(safeState),
    );
  } catch {
    // ignore storage errors (e.g. incognito quotas)
  }
}

function normalizeMessages(raw: unknown): ChatMessage[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .slice(-MAX_CHAT_MESSAGES)
    .map((message) => {
      const item = message as ChatMessage;
      const card = item.card?.chatResult
        ? {
            ...item.card,
            chatResult: {
              ...item.card.chatResult,
              answer: trimText(item.card.chatResult.answer, MAX_MESSAGE_TEXT_LENGTH),
              summary: trimText(item.card.chatResult.summary, MAX_STREAM_TEXT_LENGTH),
              answer_sections: Array.isArray(item.card.chatResult.answer_sections)
                ? item.card.chatResult.answer_sections.slice(0, 12).map((section) => ({
                    ...section,
                    source_label: trimText(section.source_label, MAX_STREAM_TEXT_LENGTH),
                    source_uri: trimText(section.source_uri ?? "", MAX_STREAM_TEXT_LENGTH),
                    summary: trimText(section.summary ?? "", MAX_STREAM_TEXT_LENGTH),
                    facts: trimTextList(section.facts),
                    open_items: trimTextList(section.open_items),
                    coding_notes: trimTextList(section.coding_notes),
                    links: trimTextList(section.links),
                    timestamps: trimTextList(section.timestamps),
                  }))
                : undefined,
              artifacts: Array.isArray(item.card.chatResult.artifacts)
                ? item.card.chatResult.artifacts
                    .slice(0, MAX_CARD_ARTIFACTS)
                    .map((artifact) => ({
                      ...artifact,
                      title: trimText(artifact.title, MAX_ARTIFACT_TEXT_LENGTH),
                      body: trimText(artifact.body, MAX_ARTIFACT_TEXT_LENGTH),
                      preview: trimText(artifact.preview, MAX_ARTIFACT_TEXT_LENGTH),
                    }))
                : [],
            },
          }
        : item.card;
      const stream = item.stream
        ? {
            ...item.stream,
            latestLine: trimText(item.stream.latestLine, MAX_STREAM_TEXT_LENGTH),
            summary: item.stream.summary
              ? trimText(item.stream.summary, MAX_STREAM_TEXT_LENGTH)
              : undefined,
            lines: Array.isArray(item.stream.lines)
              ? item.stream.lines
                  .slice(-MAX_STREAM_LINES)
                  .map((line) => trimText(line, MAX_STREAM_TEXT_LENGTH))
              : [],
          }
        : undefined;
      return {
        ...item,
        text: trimText(item.text, MAX_MESSAGE_TEXT_LENGTH),
        card,
        stream,
      };
    });
}

function loadMessages(path: string): ChatMessage[] {
  try {
    const storage = getLocalStorage();
    if (!storage) return [];
    const raw = storage.getItem(CHAT_KEY_PREFIX + path);
    if (raw) {
      const parsed = JSON.parse(raw) as unknown;
      const messages = normalizeMessages(parsed);
      const originalCount = Array.isArray(parsed) ? parsed.length : 0;
      if (originalCount !== messages.length || JSON.stringify(messages) !== raw) {
        storage.setItem(CHAT_KEY_PREFIX + path, JSON.stringify(messages));
      }
      return messages;
    }
  } catch {
    // ignore parse errors
  }
  return [];
}

function saveMessages(path: string, messages: ChatMessage[]): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    // Keep cached chat bounded so stale browser state cannot block page render.
    const trimmed = normalizeMessages(messages);
    storage.setItem(CHAT_KEY_PREFIX + path, JSON.stringify(trimmed));
  } catch {
    // ignore
  }
}

function loadWorkspacePaths(): string[] {
  const paths = new Set<string>([DEMO_WORKSPACE_PATH, DEFAULT_WORKSPACE_PATH]);
  try {
    const storage = getLocalStorage();
    if (!storage) return [...paths];
    const raw = storage.getItem(WORKSPACE_LIST_KEY);
    if (raw) {
      for (const path of JSON.parse(raw) as string[]) {
        if (path) paths.add(path);
      }
    }
    for (let i = 0; i < storage.length; i += 1) {
      const key = storage.key(i) ?? "";
      if (key.startsWith(STORAGE_KEY_PREFIX)) {
        const path = key.slice(STORAGE_KEY_PREFIX.length);
        paths.add(path);
      }
    }
  } catch {
    // ignore storage errors
  }
  return [...paths].slice(0, MAX_WORKSPACES);
}

function saveWorkspacePaths(paths: string[]): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    storage.setItem(
      WORKSPACE_LIST_KEY,
      JSON.stringify([...new Set(paths)].slice(0, MAX_WORKSPACES)),
    );
  } catch {
    // ignore storage errors
  }
}

function loadWorkspaceList(): ProjectState[] {
  return loadWorkspacePaths().map((path) => loadProject(path));
}

function rememberWorkspace(projectState: ProjectState): void {
  _workspaces.update((items) => {
    const rest = items.filter((item) => item.workspacePath !== projectState.workspacePath);
    const next = [projectState, ...rest];
    saveWorkspacePaths(next.map((item) => item.workspacePath));
    return next;
  });
}

function projectFromWorkspaceRecord(record: WorkspaceRecord): ProjectState {
  const path = cleanWorkspacePath(record.path);
  const existing = loadProject(path);
  return {
    ...existing,
    workspacePath: path,
    name: record.name || existing.name,
    createdAt: record.created_at ?? existing.createdAt,
  };
}

// ---- active project store ----

const initialPath = loadActiveWorkspacePath();
const _project = writable<ProjectState>(loadProject(initialPath));
const _messages = writable<ChatMessage[]>(loadMessages(initialPath));
const _workspaces = writable<ProjectState[]>(loadWorkspaceList());
let lastSavedProjectJSON = "";

// Sync writes to localStorage.
_project.subscribe((p) => {
  const nextJSON = JSON.stringify(p);
  if (nextJSON === lastSavedProjectJSON) return;
  lastSavedProjectJSON = nextJSON;
  saveProject(p);
  rememberWorkspace(p);
});
_messages.subscribe((m) => saveMessages(get(_project).workspacePath, m));

export const project = { subscribe: _project.subscribe };
export const chatMessages = { subscribe: _messages.subscribe };
export const workspaces = { subscribe: _workspaces.subscribe };

/** Switch to a different workspace path, loading persisted state. */
export function openProject(workspacePath: string): void {
  const cleanPath = cleanWorkspacePath(workspacePath);
  const p = loadProject(cleanPath);
  const m = loadMessages(cleanPath);
  getLocalStorage()?.setItem(ACTIVE_WORKSPACE_KEY, cleanPath);
  _project.set(p);
  _messages.set(m);
  if (cleanPath === DEMO_WORKSPACE_PATH) return;
  // Fire-and-forget: register workspace with the backend (non-blocking).
  upsertWorkspace(cleanPath, p.name).catch(() => {/* ignore offline */});
}

/** Update project name. */
export function renameProject(name: string): void {
  _project.update((p) => {
    const next = { ...p, name };
    upsertWorkspace(next.workspacePath, name).catch(() => {/* ignore offline */});
    return next;
  });
}

/** Add a workspace and switch to it. */
export function addWorkspace(workspacePath: string, name?: string): void {
  const cleanPath = workspacePath.trim();
  if (!cleanPath) return;
  const projectState = loadProject(cleanPath);
  const next = { ...projectState, name: name?.trim() || projectState.name };
  saveProject(next);
  rememberWorkspace(next);
  openProject(cleanPath);
}

/** Remove a workspace from local project, chat, source state, and the switcher. */
export function removeWorkspace(workspacePath: string): void {
  if (workspacePath === DEFAULT_WORKSPACE_PATH || workspacePath === DEMO_WORKSPACE_PATH) return;
  const storage = getLocalStorage();
  storage?.removeItem(STORAGE_KEY_PREFIX + workspacePath);
  storage?.removeItem(CHAT_KEY_PREFIX + workspacePath);
  _workspaces.update((items) => {
    const next = items.filter((item) => item.workspacePath !== workspacePath);
    saveWorkspacePaths(next.map((item) => item.workspacePath));
    return next.length > 0 ? next : [loadProject(DEFAULT_WORKSPACE_PATH)];
  });
  if (get(_project).workspacePath === workspacePath) {
    openProject(get(_workspaces)[0]?.workspacePath ?? DEFAULT_WORKSPACE_PATH);
  }
}

/** Hydrate the local workspace switcher from backend workspace records. */
export async function hydrateWorkspaces(): Promise<void> {
  const records = await getWorkspaces();
  if (records.length === 0) return;
  _workspaces.update((items) => {
    const byPath = new Map(items.map((item) => [item.workspacePath, item]));
    for (const record of records) {
      const workspace = projectFromWorkspaceRecord(record);
      byPath.set(workspace.workspacePath, workspace);
    }
    const next = [...byPath.values()];
    saveWorkspacePaths(next.map((item) => item.workspacePath));
    return next;
  });
}

/** Record a connector's knowledge state. */
export function setConnectorKnowledge(
  connector: ConnectorKind,
  uri: string,
  status: KnowledgeStatus,
  extra: Partial<Omit<ConnectorKnowledge, "connector" | "uri" | "status">> = {},
): void {
  _project.update((p) => {
    const rest = p.connectors.filter(
      (c) => !(c.connector === connector && c.uri === uri),
    );
    const next = {
      ...p,
      connectors: [
        ...rest,
        {
          connector,
          uri,
          status,
          lastIngestedAt:
            status === "ready" ? new Date().toISOString() : extra.lastIngestedAt,
          ...extra,
        },
      ],
    };
    rememberWorkspace(next);
    return next;
  });
}

/** Mark project knowledge as fully installed. */
export function markKnowledgeInstalled(): void {
  _project.update((p) => ({
    ...p,
    knowledgeInstalledAt: new Date().toISOString(),
  }));
}

/** Clear locally remembered knowledge readiness for a fresh DB-backed start. */
export function clearKnowledgeState(): void {
  _project.update((p) => {
    const next = {
      ...p,
      connectors: [],
      knowledgeInstalledAt: undefined,
    };
    rememberWorkspace(next);
    return next;
  });
}

/** Build an empty local project projection for a workspace path. */
function clearedProjectState(path: string): ProjectState {
  const cleanPath = cleanWorkspacePath(path);
  const base = loadProject(cleanPath);
  return {
    ...base,
    workspacePath: cleanPath,
    connectors: [],
    knowledgeInstalledAt: undefined,
  };
}

/**
 * Clear local source/chat/project artifacts for a single workspace path.
 *
 * Removes persisted local project/chat keys and resets in-memory state for matching
 * active workspace. Keeps workspace registry entries unless explicitly removed by caller.
 */
function clearLocalWorkspaceState(path: string): void {
  const cleanPath = cleanWorkspacePath(path);
  const activePath = get(_project).workspacePath;
  try {
    const storage = getLocalStorage();
    if (storage) {
      storage.removeItem(STORAGE_KEY_PREFIX + cleanPath);
      storage.removeItem(CHAT_KEY_PREFIX + cleanPath);
    }
  } catch {
    // ignore storage errors
  }

  _workspaces.update((items) => {
    const next = items.slice();
    const matchIndex = next.findIndex(
      (item) => item.workspacePath === cleanPath,
    );
    if (matchIndex >= 0) {
      next[matchIndex] = clearedProjectState(cleanPath);
    } else {
      next.unshift(clearedProjectState(cleanPath));
    }
    saveWorkspacePaths(next.map((item) => item.workspacePath));
    return next;
  });

  if (activePath === cleanPath) {
    _project.set(clearedProjectState(cleanPath));
  }
  if (activePath === cleanPath) {
    _messages.set([]);
  }
}

/** Clear locally remembered source and chat state for all known workspaces. */
export function clearAllLocalWorkspaceState(paths: string[] = []): void {
  const allPaths = new Set([...loadWorkspacePaths(), ...paths.map(cleanWorkspacePath)]);
  for (const path of allPaths) {
    clearLocalWorkspaceState(path);
  }

  // Keep any active workspace metadata clean even when not in any known list.
  const activePath = get(_project).workspacePath;
  const active = clearedProjectState(activePath);
  if (!allPaths.has(activePath)) {
    _project.set(active);
    _messages.set([]);
  }
}

/** Return a snapshot of the current project state. */
export function getProject(): ProjectState {
  return get(_project);
}

/**
 * Fetch workspace status from the API and update connector event counts in the store.
 * Silently no-ops when the API is unreachable.
 */
export async function loadWorkspaceStatus(workspacePath: string): Promise<WorkspaceStatus | null> {
  try {
    const status = await getWorkspaceStatus(workspacePath);
    if (!status?.syncs) return status;
    _project.update((p) => {
      const { connectors: updated, changed } = applyWorkspaceSyncsToConnectors(
        p.connectors,
        status.syncs,
      );
      if (!changed) return p;
      return { ...p, connectors: updated };
    });
    return status;
  } catch {
    return null;
  }
}

/** Add a chat message. */
export function addMessage(msg: ChatMessage): void {
  _messages.update((m) => [...m, msg]);
}

/** Replace a message by id (used to swap a loading bubble for a real response). */
export function replaceMessage(id: string, msg: ChatMessage): void {
  _messages.update((m) => m.map((x) => (x.id === id ? msg : x)));
}

/** Clear all chat history and best-effort backend chat session memory for the current project. */
export async function clearChat(): Promise<void> {
  const workspaceID = getProject().workspacePath;
  if (workspaceID && workspaceID !== DEMO_WORKSPACE_PATH) {
    await resetChatSession({ workspace_id: workspaceID });
  }
  _messages.set([]);
}
