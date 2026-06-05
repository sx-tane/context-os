import { postChatQuery, streamChatQuery } from "$lib/api";
import { demoChatQueryResult } from "$lib/chat/demoWorkspace";
import { DEMO_WORKSPACE_PATH } from "$lib/workspace/projectStore";
import type {
  Artifact,
  ChatMessage,
  ChatQueryResult,
  ChatStreamState,
  ConnectorKnowledge,
} from "$lib/types";

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
  readySources?: Pick<ConnectorKnowledge, "connector" | "uri" | "status">[];
  signal?: AbortSignal;
  isCurrent?: () => boolean;
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

export function formatAssistantResultText(result: ChatQueryResult) {
  const summary = result.summary?.trim() ?? "";
  const answer = result.answer?.trim() ?? "";
  if (!answer) return summary;
  if (!summary) return answer;

  const distinctSummary = distinctDetail(answer, summary);
  if (!distinctSummary) {
    return answer;
  }
  return `${answer}\n\n**Source summary**\n${distinctSummary}`;
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

export function streamMsg(id: string, stream: ChatStreamState, text = ""): ChatMessage {
  return {
    id,
    role: "assistant",
    text,
    createdAt: now(),
    loading: stream.status === "running",
    stream,
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
  if (requestsAllSourceConnectors(text)) {
    return "**Live Codex**\n1. Starting connected-source lookups across selected live connectors.\n2. Collecting Jira, GitHub, Slack, Google Drive, Notion, and SharePoint answers as available.\n\n**Local DB**\n1. Fallback only if no live connector answers.\n2. Uses persisted artifacts, graph, findings, and evidence history.";
  }
  const route = inferLiveRoute(text);
  if (!route.connector) {
    return "**Local DB**\n1. Classifying question against saved workspace sources.\n2. Checking persisted artifacts, graph, findings, and source registry.";
  }

  const connectorLabel = connectorDisplayName(route.connector);
  const source = route.sourceURI || "saved connector scope";
  return `**Live Codex**\n1. ${connectorLabel} plugin lookup: ${source}\n2. Running through Codex CLI via /chat/query/stream.\n\n**Local DB**\n1. Fallback only if live lookup fails or has no answer.\n2. Uses persisted artifacts, graph, findings, and evidence history.`;
}

export function localDBStatusLine(result: ChatQueryResult | null | undefined) {
  if (!result?.evidence_save_status) return "";
  switch (result.evidence_save_status) {
    case "saved":
      return `Local DB: saved ${artifactCountLabel(result.evidence_event_count ?? 0)}${result.evidence_graph_status === "updated" ? "; graph updated" : ""}`;
    case "saving":
      return "Local DB: saving source evidence...";
    case "skipped":
      return `Local DB: skipped ${skipReasonLabel(result)}`;
    case "error":
      return `Local DB: save failed${result.evidence_save_error ? `: ${result.evidence_save_error}` : ""}`;
    default:
      return `Local DB: ${result.evidence_save_status}`;
  }
}

export function appendStreamLine(stream: ChatStreamState, line: string): ChatStreamState {
  const clean = line.trim();
  if (!clean) return stream;
  return {
    ...stream,
    lines: [...stream.lines, clean],
    latestLine: clean,
    expanded: stream.expanded ?? false,
  };
}

export function buildStreamSummary(result: ChatQueryResult | null | undefined) {
  const summary = result?.summary?.trim();
  if (summary) return summary;
  return previewAnswer(result?.answer ?? "");
}

export function isNearBottom(
  element: Pick<HTMLElement, "scrollHeight" | "scrollTop" | "clientHeight">,
  threshold = 96,
) {
  return element.scrollHeight - element.scrollTop - element.clientHeight <= threshold;
}

export async function runChatQuery(options: ChatQueryOptions) {
  const loadText = buildChatLoadingText(options.text);
  const route = inferLiveRoute(options.text);
  const initialStream = initialStreamState(loadText);
  const load = streamMsg(makeId(), initialStream);
  options.addMessage(load);
  options.setBusy(true);
  let refreshedWorkspace = false;
  let stream = initialStream;
  let streamedAnswerText = "";
  const isCurrentRun = () => options.isCurrent?.() ?? true;
  const replaceIfCurrent = (message: ChatMessage) => {
    if (!isCurrentRun()) return false;
    options.replaceMessage(load.id, message);
    return true;
  };
  const stopIfCurrent = () => {
    if (!isCurrentRun()) return;
    stream = stoppedStream(stream);
    replaceIfCurrent({
      ...streamMsg(load.id, stream, streamedAnswerText),
      loading: false,
    });
  };
  try {
    if (options.workspacePath === DEMO_WORKSPACE_PATH) {
      const result = demoChatQueryResult(options.text);
      options.setLastChatResult(result);
      options.setActivityArtifacts(result.artifacts);
      replaceIfCurrent(
        assistantMsg(formatAssistantResultText(result), {
          kind: "query",
          chatResult: result,
        }),
      );
      return;
    }

    const useConnectorFanout = requestsAllSourceConnectors(options.text);
    const body = {
      workspace_id: options.workspacePath,
      message: options.text,
      ...(!useConnectorFanout && route.connector ? { connector: route.connector } : {}),
      ...((useConnectorFanout || (!route.connector && !route.sourceURI))
        ? liveConnectorHint(options.readySources)
        : {}),
      ...(!useConnectorFanout && route.sourceURI ? { source_uri: route.sourceURI } : {}),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      local_date: localDateString(new Date()),
      response_language: detectResponseLanguage(options.text),
      limit: 20,
    };
    let streamedResult: ChatQueryResult | null = null;
    let streamedAnswer: ChatQueryResult | null = null;
    let streamStarted = false;
    try {
      await streamChatQuery(
        body,
        {
          onLog: (line) => {
            if (!isCurrentRun()) return;
            streamStarted = true;
            stream = appendStreamLine(stream, line);
            replaceIfCurrent(streamMsg(load.id, stream, streamedAnswerText));
          },
          onStatus: (elapsed) => {
            if (!isCurrentRun()) return;
            streamStarted = true;
            const statusLine = `• Codex still running... ${elapsed}s`;
            stream = replaceStreamStatusLine(stream, statusLine);
            replaceIfCurrent(streamMsg(load.id, stream, streamedAnswerText));
          },
          onAnswer: (result) => {
            if (!isCurrentRun()) return;
            streamedAnswer = result;
            streamedAnswerText = formatAssistantResultText(result);
            options.setLastChatResult(result);
            stream = { ...stream, summary: buildStreamSummary(result) };
            replaceIfCurrent(streamMsg(load.id, stream, streamedAnswerText));
          },
          onResult: (result) => {
            if (!isCurrentRun()) return;
            streamedResult = result;
          },
          onError: (message) => {
            throw new Error(message);
          },
        },
        { signal: options.signal },
      );
    } catch (error) {
      if (isCanceledQuery(error, options.signal)) {
        stopIfCurrent();
        return;
      }
      if (!isCurrentRun() || options.signal?.aborted) {
        stopIfCurrent();
        return;
      }
      const answeredResult = streamedAnswer as ChatQueryResult | null;
      if (answeredResult) {
        const failedResult: ChatQueryResult = {
          ...answeredResult,
          evidence_save_status: "error",
          evidence_save_error: errorMessage(error),
        };
        options.setLastChatResult(failedResult);
        stream = appendStreamLine(stream, `• Local DB save failed: ${errorMessage(error)}`);
        replaceIfCurrent(
          {
            ...assistantMsg(formatAssistantResultText(failedResult), {
              kind: "query",
              chatResult: failedResult,
            }),
            id: load.id,
            stream: {
              ...stream,
              status: "complete",
              summary: finalStreamSummary(failedResult),
              expanded: stream.expanded ?? false,
            },
          },
        );
        return;
      }
      if (streamStarted) {
        stream = {
          ...appendStreamLine(stream, `• Live Codex lookup failed: ${errorMessage(error)}`),
          status: "error",
          summary: buildStreamSummary(streamedAnswer) || "Live Codex lookup failed.",
        };
        replaceIfCurrent({
          ...streamMsg(load.id, stream, streamedAnswerText),
          loading: false,
        });
        return;
      }
      stream = appendStreamLine(stream, `• Streaming unavailable: ${errorMessage(error)}`);
      stream = appendStreamLine(stream, "• Falling back to standard chat query.");
      replaceIfCurrent(streamMsg(load.id, stream, streamedAnswerText));
    }
    if (!isCurrentRun() || options.signal?.aborted) {
      stopIfCurrent();
      return;
    }
    const finalStreamedResult = streamedResult as ChatQueryResult | null;
    if (finalStreamedResult) {
      if (!isCurrentRun()) return;
      options.setLastChatResult(finalStreamedResult);
      if (finalStreamedResult.artifacts?.length) {
        options.setActivityArtifacts(finalStreamedResult.artifacts);
      }
      replaceIfCurrent(
        {
          ...assistantMsg(formatAssistantResultText(finalStreamedResult), {
            kind: "query",
            chatResult: finalStreamedResult,
          }),
          id: load.id,
          stream: {
            ...stream,
            status: "complete",
            summary: finalStreamSummary(finalStreamedResult),
            expanded: stream.expanded ?? false,
          },
        },
      );
      if (finalStreamedResult.evidence_save_status === "saved") {
        await options.refreshWorkspace();
        refreshedWorkspace = true;
      }
      return;
    }

    if (!isCurrentRun() || options.signal?.aborted) {
      stopIfCurrent();
      return;
    }
    const res = await postChatQuery(body, { signal: options.signal });
    if (!isCurrentRun() || options.signal?.aborted) {
      stopIfCurrent();
      return;
    }
    if (res.ok) {
      options.setLastChatResult(res.body);
      if (res.body.artifacts?.length) {
        options.setActivityArtifacts(res.body.artifacts);
      }
      replaceIfCurrent(
        {
          ...assistantMsg(formatAssistantResultText(res.body), {
            kind: "query",
            chatResult: res.body,
          }),
          id: load.id,
          stream: {
            ...stream,
            status: "complete",
            summary: finalStreamSummary(res.body),
            expanded: stream.expanded ?? false,
          },
        },
      );
      return;
    }
    replaceIfCurrent(
      assistantMsg(
        `Source query failed: ${res.body.message ?? res.body.error ?? "unknown error"}`,
      ),
    );
  } catch (error) {
    if (isCanceledQuery(error, options.signal)) {
      stopIfCurrent();
      return;
    }
    if (!isCurrentRun()) return;
    replaceIfCurrent(
      assistantMsg(`Source query failed: ${String(error)}`),
    );
  } finally {
    options.setBusy(false);
    if (!refreshedWorkspace) {
      await options.refreshWorkspace();
    }
  }
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function isAbortError(error: unknown) {
  return (
    typeof error === "object" &&
    error !== null &&
    "name" in error &&
    error.name === "AbortError"
  );
}

function isCanceledQuery(error: unknown, signal?: AbortSignal) {
  if (signal?.aborted) return true;
  if (isAbortError(error)) return true;
  const message = errorMessage(error).toLowerCase();
  return message.includes("query_canceled") ||
    message.includes("context canceled") ||
    message.includes("aborted");
}

function stoppedStream(stream: ChatStreamState): ChatStreamState {
  return {
    ...appendStreamLine(stream, "• Codex chat stopped."),
    status: "complete",
    summary: "Stopped by user.",
    expanded: stream.expanded ?? false,
  };
}

export function localDateString(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function detectResponseLanguage(text: string) {
  if (/[\uac00-\ud7af]/.test(text)) return "ko";
  if (/[\u3040-\u30ff]/.test(text)) return "ja";
  const cjkCount = (text.match(/[\u4e00-\u9fff]/g) ?? []).length;
  if (cjkCount === 0) return "en";
  const englishWordCount = (text.match(/[A-Za-z][A-Za-z0-9_-]*/g) ?? []).length;
  const chineseCueCount = (
    text.match(/[吗呢吧啊的了是有和在请问中文回答最近变化什么怎么为什么]/g) ?? []
  ).length;
  if (englishWordCount >= 3 && chineseCueCount === 0 && cjkCount <= 6) return "en";
  return "zh";
}

function initialStreamState(text: string): ChatStreamState {
  const lines = text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  return {
    lines,
    latestLine: lines.at(-1) ?? "Starting source query.",
    status: "running",
    expanded: false,
  };
}

function replaceStreamStatusLine(stream: ChatStreamState, line: string): ChatStreamState {
  const lines = stream.lines.filter((item) => !item.startsWith("• Codex still running..."));
  return {
    ...stream,
    lines: [...lines, line],
    latestLine: line,
    expanded: stream.expanded ?? false,
  };
}

function previewAnswer(answer: string) {
  const clean = answer.trim().replace(/\s+/g, " ");
  if (!clean) return "";
  return clean.length > 160 ? `${clean.slice(0, 157)}...` : clean;
}

function distinctDetail(summary: string, answer: string) {
  if (!answer) return "";
  if (normalizeText(answer) === normalizeText(summary)) return "";

  const trimmedAnswer = answer.trim();
  const trimmedSummary = summary.trim();
  if (normalizeText(trimmedAnswer).startsWith(normalizeText(trimmedSummary))) {
    const remainder = trimmedAnswer.slice(trimmedSummary.length).trim();
    return remainder.replace(/^[\s:.;,-]+/, "").trim();
  }
  return trimmedAnswer;
}

function normalizeText(text: string) {
  return text.trim().replace(/\s+/g, " ").toLowerCase();
}

function finalStreamSummary(result: ChatQueryResult) {
  return localDBStatusLine(result) || buildStreamSummary(result);
}

function artifactCountLabel(count: number) {
  return `${count} artifact${count === 1 ? "" : "s"}`;
}

function skipReasonLabel(result: ChatQueryResult) {
  const connector = result.connector?.trim();
  const sourceURI = result.source_uri?.trim();
  if (connector && sourceURI && connector.toLowerCase() === sourceURI.toLowerCase()) {
    return `broad ${connector} scope`;
  }
  if (result.provider !== "codex") return "local-only answer";
  return "evidence save";
}

export function liveConnectorHint(
  readySources: Pick<ConnectorKnowledge, "connector" | "uri" | "status">[] | undefined,
) {
  const connectors = readyLiveConnectors(readySources);
  return connectors.length > 0 ? { connectors } : {};
}

function requestsAllSourceConnectors(text: string) {
  const lower = text.toLowerCase();
  return /all\s+(source\s+)?con(?:nec|en)c?tors?/.test(lower) ||
    /all\s+source/.test(lower) ||
    /check\s+all\s+source/.test(lower) ||
    /every\s+(source\s+)?con(?:nec|en)c?tors?/.test(lower) ||
    /\b(?:allow|allowed|connected|selected|enabled|available)\s+(?:source\s+)?con(?:nec|en)c?tors?\b/.test(lower) ||
    /\b(?:my|workspace)\s+(?:source\s+)?con(?:nec|en)c?tors?\b/.test(lower);
}

function readyLiveConnectors(
  readySources: Pick<ConnectorKnowledge, "connector" | "uri" | "status">[] | undefined,
) {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const source of readySources ?? []) {
    if (source.status !== "ready") continue;
    if (source.connector === "filesystem") continue;
    if (seen.has(source.connector)) continue;
    seen.add(source.connector);
    out.push(source.connector);
  }
  return out;
}

type PendingLiveRoute = {
  connector: string;
  sourceURI: string;
};

function inferLiveRoute(text: string): PendingLiveRoute {
  if (hasDriveFileName(text)) {
    return { connector: "googledrive", sourceURI: "" };
  }
  const sourceURI = inferSourceURI(text);
  const connector = inferConnector(text, sourceURI);
  return { connector, sourceURI };
}

function hasDriveFileName(text: string) {
  return /\.(xlsx|xls|gsheet|gdoc|gslides|docx|pptx)\b/i.test(text);
}

function inferSourceURI(text: string) {
  const jiraKey = text.match(/\b[A-Z][A-Z0-9]+-\d+\b/);
  if (jiraKey) return jiraKey[0];
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
    if (/^[A-Z][A-Z0-9]+-\d+$/.test(sourceURI)) return "jira";
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
