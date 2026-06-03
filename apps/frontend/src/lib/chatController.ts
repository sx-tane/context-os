import { postChatQuery } from "$lib/api";
import { demoChatQueryResult } from "$lib/demoWorkspace";
import { DEMO_WORKSPACE_PATH } from "$lib/projectStore";
import type { Artifact, ChatMessage, ChatQueryResult } from "$lib/types";

export type ChatCommandAction =
  | "clear"
  | "openSources"
  | "runFindings"
  | "query";

export type ChatQueryOptions = {
  text: string;
  workspacePath: string;
  addMessage: (message: ChatMessage) => void;
  replaceMessage: (id: string, message: ChatMessage) => void;
  setBusy: (busy: boolean) => void;
  setLastChatResult: (result: ChatQueryResult | null) => void;
  setActivityArtifacts: (artifacts: Artifact[]) => void;
  refreshWorkspace: () => Promise<void>;
};

export function makeId() {
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}

export function now() {
  return new Date().toISOString();
}

export function userMsg(text: string): ChatMessage {
  return { id: makeId(), role: "user", text, createdAt: now() };
}

export function assistantMsg(
  text: string,
  card?: ChatMessage["card"],
): ChatMessage {
  return {
    id: makeId(),
    role: "assistant",
    text,
    createdAt: now(),
    card,
  };
}

export function loadingMsg(text: string): ChatMessage {
  return {
    id: makeId(),
    role: "assistant",
    text,
    createdAt: now(),
    loading: true,
  };
}

export function progressMsg(id: string, text: string): ChatMessage {
  return {
    id,
    role: "assistant",
    text,
    createdAt: now(),
    loading: true,
  };
}

export function classifyChatCommand(text: string): ChatCommandAction {
  const lower = text.toLowerCase();
  if (lower === "clear") return "clear";
  if (
    lower.includes("install") ||
    lower.includes("setup") ||
    lower.includes("add source") ||
    lower.includes("connect source")
  ) {
    return "openSources";
  }
  if (
    lower.includes("finding") ||
    lower.includes("mismatch") ||
    lower.startsWith("analyze") ||
    lower.startsWith("analyse")
  ) {
    return "runFindings";
  }
  return "query";
}

export async function runChatQuery(options: ChatQueryOptions) {
  const load = loadingMsg("Checking connected source context...");
  options.addMessage(load);
  options.setBusy(true);
  try {
    if (options.workspacePath === DEMO_WORKSPACE_PATH) {
      const result = demoChatQueryResult(options.text);
      options.setLastChatResult(result);
      options.setActivityArtifacts(result.artifacts);
      options.replaceMessage(
        load.id,
        assistantMsg(result.answer, {
          kind: "query",
          chatResult: result,
        }),
      );
      return;
    }

    const res = await postChatQuery({
      workspace_id: options.workspacePath,
      message: options.text,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      local_date: localDateString(new Date()),
      limit: 20,
    });
    if (res.ok) {
      options.setLastChatResult(res.body);
      options.replaceMessage(
        load.id,
        assistantMsg(res.body.answer, {
          kind: "query",
          chatResult: res.body,
        }),
      );
      return;
    }
    options.replaceMessage(
      load.id,
      assistantMsg(
        `Source query failed: ${res.body.message ?? res.body.error ?? "unknown error"}`,
      ),
    );
  } catch (error) {
    options.replaceMessage(
      load.id,
      assistantMsg(`Source query failed: ${String(error)}`),
    );
  } finally {
    options.setBusy(false);
    await options.refreshWorkspace();
  }
}

export function localDateString(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
