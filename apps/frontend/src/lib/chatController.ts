import { postChatQuery, streamChatQuery } from "$lib/api";
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

export function buildChatLoadingText(text: string) {
  const route = inferLiveRoute(text);
  if (!route.connector) {
    return "**Local DB**\n1. Classifying question against saved workspace sources.\n2. Checking persisted artifacts, graph, findings, and source registry.";
  }

  const connectorLabel = connectorDisplayName(route.connector);
  const source = route.sourceURI || "saved connector scope";
  return `**Live Codex**\n1. ${connectorLabel} plugin lookup: ${source}\n2. Running through Codex CLI via /chat/query/stream.\n\n**Local DB**\n1. Fallback only if live lookup fails or has no answer.\n2. Uses persisted artifacts, graph, findings, and evidence history.`;
}

export async function runChatQuery(options: ChatQueryOptions) {
  const load = loadingMsg(buildChatLoadingText(options.text));
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

    const body = {
      workspace_id: options.workspacePath,
      message: options.text,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      local_date: localDateString(new Date()),
      limit: 20,
    };
    let streamedResult: ChatQueryResult | null = null;
    let streamStarted = false;
    const transcript: string[] = [load.text];
    try {
      await streamChatQuery(body, {
        onLog: (line) => {
          streamStarted = true;
          appendTranscriptLine(transcript, line);
          options.replaceMessage(load.id, progressMsg(load.id, transcript.join("\n")));
        },
        onStatus: (elapsed) => {
          streamStarted = true;
          const statusLine = `• Codex still running... ${elapsed}s`;
          const next = [...transcript.filter((line) => !line.startsWith("• Codex still running...")), statusLine];
          transcript.splice(0, transcript.length, ...next);
          options.replaceMessage(load.id, progressMsg(load.id, transcript.join("\n")));
        },
        onResult: (result) => {
          streamedResult = result;
        },
        onError: (message) => {
          throw new Error(message);
        },
      });
    } catch (error) {
      if (streamStarted) {
        appendTranscriptLine(transcript, `• Live Codex lookup failed: ${errorMessage(error)}`);
        options.replaceMessage(load.id, assistantMsg(transcript.join("\n")));
        return;
      }
      appendTranscriptLine(transcript, `• Streaming unavailable: ${errorMessage(error)}`);
      appendTranscriptLine(transcript, "• Falling back to standard chat query.");
      options.replaceMessage(load.id, progressMsg(load.id, transcript.join("\n")));
    }
    const finalStreamedResult = streamedResult as ChatQueryResult | null;
    if (finalStreamedResult) {
      options.setLastChatResult(finalStreamedResult);
      options.replaceMessage(
        load.id,
        assistantMsg(finalStreamedResult.answer, {
          kind: "query",
          chatResult: finalStreamedResult,
        }),
      );
      return;
    }

    const res = await postChatQuery(body);
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

function appendTranscriptLine(lines: string[], line: string) {
  const clean = line.trim();
  if (!clean) return;
  lines.push(clean);
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

export function localDateString(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

type PendingLiveRoute = {
  connector: string;
  sourceURI: string;
};

function inferLiveRoute(text: string): PendingLiveRoute {
  const sourceURI = inferSourceURI(text);
  const connector = inferConnector(text, sourceURI);
  return { connector, sourceURI };
}

function inferSourceURI(text: string) {
  const match = text.match(/(#[A-Za-z0-9_.-]+|[a-z]+:\/\/[^\s,]+|https?:\/\/[^\s,]+|[A-Za-z0-9_.-]+\/[A-Za-z0-9_./-]+)/i);
  return match?.[0]?.replace(/[.,;:"'()[\]{}]+$/g, "") ?? "";
}

function inferConnector(text: string, sourceURI: string) {
  const lower = text.toLowerCase();
  if (
    lower.includes("filesystem") ||
    lower.includes("file system") ||
    lower.includes("local file") ||
    lower.includes("docs/")
  ) {
    return "";
  }
  if (sourceURI) {
    const source = sourceURI.toLowerCase();
    if (source.startsWith("#") || source.startsWith("slack://") || source.includes("slack.com")) return "slack";
    if (source.startsWith("github://") || source.includes("github.com") || source.includes("api.github.com")) return "github";
    if (/^[a-z0-9_.-]+\/[a-z0-9_.-]+$/i.test(sourceURI)) return "github";
    if (source.startsWith("jira://") || source.includes("atlassian.net") || source.includes("/browse/")) return "jira";
    if (source.startsWith("notion://") || source.includes("notion.so") || source.includes("notion.site")) return "notion";
    if (source.startsWith("googledrive://") || source.startsWith("gdrive://") || source.includes("drive.google.com") || source.includes("docs.google.com")) return "googledrive";
    if (source.startsWith("sharepoint://") || source.includes("sharepoint.com") || source.includes("onedrive.live.com")) return "sharepoint";
  }
  if (lower.includes("google drive") || lower.includes("googledrive") || lower.includes("gdrive")) return "googledrive";
  if (lower.includes("sharepoint") || lower.includes("one drive") || lower.includes("onedrive")) return "sharepoint";
  if (lower.includes("github") || lower.includes("pull request") || lower.includes("pr ") || lower.includes("repo") || lower.includes("commit")) return "github";
  if (lower.includes("slack") || lower.includes("channel")) return "slack";
  if (lower.includes("jira") || lower.includes("ticket") || lower.includes("issue") || lower.includes("sprint")) return "jira";
  if (lower.includes("notion")) return "notion";
  return "";
}

function connectorDisplayName(connector: string) {
  const labels: Record<string, string> = {
    github: "GitHub",
    jira: "Jira/Rovo",
    slack: "Slack",
    notion: "Notion",
    googledrive: "Google Drive",
    sharepoint: "SharePoint",
  };
  return labels[connector] ?? connector;
}
