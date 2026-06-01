import { writable, get } from "svelte/store";
import type {
  ChatMessage,
  ConnectorKind,
  ConnectorKnowledge,
  KnowledgeStatus,
  ProjectState,
  WorkspaceRecord,
} from "$lib/types";
import { getWorkspaces, upsertWorkspace, getWorkspaceStatus } from "$lib/api";

const STORAGE_KEY_PREFIX = "contextos_project_";
const CHAT_KEY_PREFIX = "contextos_chat_";
const WORKSPACE_LIST_KEY = "contextos_workspaces";
const ACTIVE_WORKSPACE_KEY = "contextos_workspace_path";

// Default workspace path — replaced by user selection or URL param.
const DEFAULT_PATH = "/workspace";

function getLocalStorage(): Storage | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage;
}

function loadActiveWorkspacePath(): string {
  return getLocalStorage()?.getItem(ACTIVE_WORKSPACE_KEY) || DEFAULT_PATH;
}

function cleanWorkspacePath(path?: string): string {
  const cleanPath = path?.trim();
  return cleanPath || DEFAULT_PATH;
}

function loadProject(path: string): ProjectState {
  const cleanPath = cleanWorkspacePath(path);
  try {
    const storage = getLocalStorage();
    if (!storage) return defaultProject(cleanPath);
    const raw = storage.getItem(STORAGE_KEY_PREFIX + cleanPath);
    if (raw) return JSON.parse(raw) as ProjectState;
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

function saveProject(state: ProjectState): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    storage.setItem(
      STORAGE_KEY_PREFIX + state.workspacePath,
      JSON.stringify(state),
    );
  } catch {
    // ignore storage errors (e.g. incognito quotas)
  }
}

function loadMessages(path: string): ChatMessage[] {
  try {
    const storage = getLocalStorage();
    if (!storage) return [];
    const raw = storage.getItem(CHAT_KEY_PREFIX + path);
    if (raw) return JSON.parse(raw) as ChatMessage[];
  } catch {
    // ignore parse errors
  }
  return [];
}

function saveMessages(path: string, messages: ChatMessage[]): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    // Keep last 200 messages to avoid storage bloat.
    const trimmed = messages.slice(-200);
    storage.setItem(CHAT_KEY_PREFIX + path, JSON.stringify(trimmed));
  } catch {
    // ignore
  }
}

function loadWorkspacePaths(): string[] {
  const paths = new Set<string>([DEFAULT_PATH]);
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
        paths.add(key.slice(STORAGE_KEY_PREFIX.length));
      }
    }
  } catch {
    // ignore storage errors
  }
  return [...paths];
}

function saveWorkspacePaths(paths: string[]): void {
  try {
    const storage = getLocalStorage();
    if (!storage) return;
    storage.setItem(WORKSPACE_LIST_KEY, JSON.stringify([...new Set(paths)]));
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

// Sync writes to localStorage.
_project.subscribe((p) => {
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

/** Remove a workspace from the local switcher without deleting backend data. */
export function removeWorkspace(workspacePath: string): void {
  _workspaces.update((items) => {
    const next = items.filter((item) => item.workspacePath !== workspacePath);
    saveWorkspacePaths(next.map((item) => item.workspacePath));
    return next.length > 0 ? next : [loadProject(DEFAULT_PATH)];
  });
  if (get(_project).workspacePath === workspacePath) {
    openProject(get(_workspaces)[0]?.workspacePath ?? DEFAULT_PATH);
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
    const rest = p.connectors.filter((c) => c.connector !== connector);
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

/** Return a snapshot of the current project state. */
export function getProject(): ProjectState {
  return get(_project);
}

/**
 * Fetch workspace status from the API and update connector event counts in the store.
 * Silently no-ops when the API is unreachable.
 */
export async function loadWorkspaceStatus(workspacePath: string): Promise<void> {
  try {
    const status = await getWorkspaceStatus(workspacePath);
    if (!status?.syncs) return;
    _project.update((p) => {
      const updated = p.connectors.map((ck) => {
        const sync = status.syncs?.find((s) => s.connector === ck.connector);
        if (!sync) return ck;
        return { ...ck, eventCount: sync.event_count ?? ck.eventCount };
      });
      return { ...p, connectors: updated };
    });
  } catch {
    // ignore when backend is offline
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

/** Clear all chat history for the current project. */
export function clearChat(): void {
  _messages.set([]);
}
