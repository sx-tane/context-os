import { writable, get } from "svelte/store";
import type {
  ChatMessage,
  ConnectorKind,
  ConnectorKnowledge,
  KnowledgeStatus,
  ProjectState,
} from "$lib/types";

const STORAGE_KEY_PREFIX = "contextos_project_";
const CHAT_KEY_PREFIX = "contextos_chat_";

// Default workspace path — replaced by user selection or URL param.
const DEFAULT_PATH = "/workspace";

function loadProject(path: string): ProjectState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_PREFIX + path);
    if (raw) return JSON.parse(raw) as ProjectState;
  } catch {
    // ignore parse errors
  }
  return {
    workspacePath: path,
    name: path.split("/").filter(Boolean).pop() ?? "project",
    createdAt: new Date().toISOString(),
    connectors: [],
  };
}

function saveProject(state: ProjectState): void {
  try {
    localStorage.setItem(
      STORAGE_KEY_PREFIX + state.workspacePath,
      JSON.stringify(state),
    );
  } catch {
    // ignore storage errors (e.g. incognito quotas)
  }
}

function loadMessages(path: string): ChatMessage[] {
  try {
    const raw = localStorage.getItem(CHAT_KEY_PREFIX + path);
    if (raw) return JSON.parse(raw) as ChatMessage[];
  } catch {
    // ignore parse errors
  }
  return [];
}

function saveMessages(path: string, messages: ChatMessage[]): void {
  try {
    // Keep last 200 messages to avoid storage bloat.
    const trimmed = messages.slice(-200);
    localStorage.setItem(CHAT_KEY_PREFIX + path, JSON.stringify(trimmed));
  } catch {
    // ignore
  }
}

// ---- active project store ----

const _project = writable<ProjectState>(loadProject(DEFAULT_PATH));
const _messages = writable<ChatMessage[]>(loadMessages(DEFAULT_PATH));

// Sync writes to localStorage.
_project.subscribe((p) => saveProject(p));
_messages.subscribe((m) => saveMessages(get(_project).workspacePath, m));

export const project = { subscribe: _project.subscribe };
export const chatMessages = { subscribe: _messages.subscribe };

/** Switch to a different workspace path, loading persisted state. */
export function openProject(workspacePath: string): void {
  const p = loadProject(workspacePath);
  const m = loadMessages(workspacePath);
  _project.set(p);
  _messages.set(m);
}

/** Update project name. */
export function renameProject(name: string): void {
  _project.update((p) => ({ ...p, name }));
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
    return {
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
  });
}

/** Mark project knowledge as fully installed. */
export function markKnowledgeInstalled(): void {
  _project.update((p) => ({
    ...p,
    knowledgeInstalledAt: new Date().toISOString(),
  }));
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

/** Return the current project snapshot without subscribing. */
export function getProject(): ProjectState {
  return get(_project);
}
