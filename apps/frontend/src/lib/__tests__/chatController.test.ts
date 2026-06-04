jest.mock("$lib/api");
jest.mock("$lib/workspace/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  appendStreamLine,
  assistantMsg,
  buildChatLoadingText,
  buildStreamSummary,
  classifyChatCommand,
  detectResponseLanguage,
  isNearBottom,
  localDateString,
  localDBStatusLine,
  liveConnectorHint,
  runChatQuery,
  userMsg,
} from "../chat/controller";

import { postChatQuery, streamChatQuery } from "$lib/api";
import type { ChatQueryResult } from "$lib/types";

const mockPostChatQuery = postChatQuery as jest.Mock;
const mockStreamChatQuery = streamChatQuery as jest.Mock;

beforeEach(() => {
  mockPostChatQuery.mockReset();
  mockStreamChatQuery.mockReset();
});

type ChatControllerTestState = {
  busyCalls: boolean[];
  lastChatResult: ChatQueryResult | null;
  replacements: ChatMessageSnapshot[];
  addMessage: jest.Mock;
  replaceMessage: jest.Mock;
  setBusy: jest.Mock;
  setLastChatResult: jest.Mock;
  setActivityArtifacts: jest.Mock;
  refreshWorkspace: jest.Mock<Promise<void>, []>;
};

type ChatMessageSnapshot = {
  text: string;
  loading?: boolean;
  stream?: {
    lines: string[];
    latestLine: string;
    status: string;
    summary?: string;
  };
  card?: unknown;
};

describe("classifyChatCommand", () => {
  it("routes clear, source setup, analysis, and source query prompts", () => {
    expect(classifyChatCommand("clear")).toBe("clear");
    expect(classifyChatCommand("connect source")).toBe("openSources");
    expect(classifyChatCommand("analyze this workspace")).toBe("runFindings");
    expect(classifyChatCommand("what changed today?")).toBe("query");
  });
});

describe("chat message factories", () => {
  it("creates user and assistant messages with expected roles and cards", () => {
    expect(userMsg("hello")).toMatchObject({ role: "user", text: "hello" });
    expect(assistantMsg("done", { kind: "status" })).toMatchObject({
      role: "assistant",
      text: "done",
      card: { kind: "status" },
    });
  });
});

describe("localDateString", () => {
  it("formats dates as local yyyy-mm-dd", () => {
    expect(localDateString(new Date(2026, 0, 5))).toBe("2026-01-05");
  });
});

describe("detectResponseLanguage", () => {
  it("matches Chinese, Japanese, Korean, and English prompt language", () => {
    expect(detectResponseLanguage("请用中文回答")).toBe("zh");
    expect(detectResponseLanguage("帳票項目のマッピング確認")).toBe("ja");
    expect(detectResponseLanguage("최근 메시지 확인")).toBe("ko");
    expect(detectResponseLanguage("check recent activity")).toBe("en");
  });

  it("uses the non-English script in mixed-language prompts", () => {
    expect(detectResponseLanguage("BKGDEV-8096 帳票項目のマッピング確認.xlsx Jira Slack")).toBe("ja");
    expect(detectResponseLanguage("GitHub 最近有什么变化")).toBe("zh");
  });

  it("keeps English for English questions that include short CJK source terms", () => {
    expect(detectResponseLanguage("what about kkg payment 決済GW linkedFlag")).toBe("en");
    expect(detectResponseLanguage("why does BKGDEV-8236 mention 決済GW")).toBe("en");
  });
});

describe("buildChatLoadingText", () => {
  it("shows live Codex route for owner/repo prompts", () => {
    const text = buildChatLoadingText("help me check sx-tane/context-os");

    expect(text).toContain("Live Codex");
    expect(text).toContain("GitHub plugin lookup: sx-tane/context-os");
    expect(text).toContain("Codex CLI");
    expect(text).toContain("Local DB");
  });

  it("shows live Codex route for Jira issue keys without saying Jira", () => {
    const text = buildChatLoadingText("BKGDEV-8466 check this");

    expect(text).toContain("Live Codex");
    expect(text).toContain("Jira/Rovo plugin lookup: BKGDEV-8466");
    expect(text).toContain("Local DB");
  });

  it("shows local DB route for local file prompts", () => {
    const text = buildChatLoadingText("summarize local file docs/plan.md");

    expect(text).toContain("Local DB");
    expect(text).not.toContain("Live Codex");
  });
});

describe("stream helpers", () => {
  it("appends clean stream lines and tracks the latest line", () => {
    const next = appendStreamLine(
      { lines: ["start"], latestLine: "start", status: "running" },
      "  • done  ",
    );

    expect(next.lines).toEqual(["start", "• done"]);
    expect(next.latestLine).toBe("• done");
    expect(next.status).toBe("running");
  });

  it("builds stream summaries from explicit summaries or answer previews", () => {
    expect(buildStreamSummary(makeChatResult({ summary: "Explicit summary" }))).toBe("Explicit summary");
    expect(buildStreamSummary(makeChatResult({ summary: "", answer: "Answer text" }))).toBe("Answer text");
  });

  it("builds compact Local DB status lines for evidence saves", () => {
    expect(localDBStatusLine(makeChatResult({ evidence_save_status: "saved", evidence_event_count: 1 }))).toBe("Local DB: saved 1 artifact");
    expect(localDBStatusLine(makeChatResult({ evidence_save_status: "saved", evidence_event_count: 2, evidence_graph_status: "updated" }))).toBe("Local DB: saved 2 artifacts; graph updated");
    expect(localDBStatusLine(makeChatResult({ connector: "jira", source_uri: "jira", evidence_save_status: "skipped" }))).toBe("Local DB: skipped broad jira scope");
    expect(localDBStatusLine(makeChatResult({ evidence_save_status: "error", evidence_save_error: "offline" }))).toBe("Local DB: save failed: offline");
  });

  it("reports whether a scroll pane is near the bottom", () => {
    expect(isNearBottom({ scrollHeight: 1000, scrollTop: 820, clientHeight: 120 } as HTMLElement)).toBe(true);
    expect(isNearBottom({ scrollHeight: 1000, scrollTop: 700, clientHeight: 120 } as HTMLElement)).toBe(false);
  });
});

describe("liveConnectorHint", () => {
  it("returns unique ready live connectors and skips filesystem sources", () => {
    expect(liveConnectorHint([
      { connector: "jira", uri: "jira", status: "ready" },
      { connector: "github", uri: "sx-tane/context-os", status: "ready" },
      { connector: "github", uri: "github", status: "ready" },
      { connector: "filesystem", uri: "docs", status: "ready" },
      { connector: "slack", uri: "slack", status: "ingesting" },
    ])).toEqual({ connectors: ["jira", "github"] });
  });
});

describe("runChatQuery", () => {
  it("falls back to standard source queries when streaming is unavailable", async () => {
    mockStreamChatQuery.mockRejectedValue(new Error("stream route unavailable"));
    mockPostChatQuery.mockResolvedValue({
      ok: true,
      body: {
        intent: "status",
        workspace_id: "workspace",
        workspace_path: "workspace",
        provider: "local",
        answer: "Answer",
        summary: "Summary",
        artifact_count: 0,
        artifacts: [],
      },
    });
    const state = makeState();

    await runChatQuery({
      text: "status",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockStreamChatQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        workspace_id: "workspace",
        message: "status",
        response_language: "en",
      }),
      expect.objectContaining({
        onAnswer: expect.any(Function),
        onLog: expect.any(Function),
        onStatus: expect.any(Function),
        onResult: expect.any(Function),
        onError: expect.any(Function),
      }),
    );
    expect(mockPostChatQuery).toHaveBeenCalledWith(
      expect.objectContaining({ workspace_id: "workspace", message: "status" }),
    );
    expect(state.busyCalls).toEqual([true, false]);
    expect(state.addMessage.mock.calls[0][0].text).toBe("");
    expect(state.addMessage.mock.calls[0][0].stream.latestLine).toContain("source registry");
    expect(state.replacements[0].stream?.latestLine).toContain("standard chat query");
    expect(state.replacements[0].stream?.lines.join("\n")).toContain("Streaming unavailable");
    expect(state.lastChatResult?.answer).toBe("Answer");
    expect(state.replacements.at(-1)?.text).toBe("Answer");
    expect(state.replacements.at(-1)?.stream?.status).toBe("complete");
    expect(state.replacements.at(-1)?.stream?.summary).toBe("Summary");
    expect(state.replacements.at(-1)?.card).toBeDefined();
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });

  it("sends the detected response language with streamed and fallback queries", async () => {
    mockStreamChatQuery.mockRejectedValue(new Error("stream route unavailable"));
    mockPostChatQuery.mockResolvedValue({
      ok: true,
      body: makeChatResult({ answer: "中文回答", summary: "中文回答" }),
    });
    const state = makeState();

    await runChatQuery({
      text: "请用中文回答 GitHub 最近有什么变化",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockStreamChatQuery.mock.calls[0][0]).toMatchObject({
      response_language: "zh",
    });
    expect(mockPostChatQuery.mock.calls[0][0]).toMatchObject({
      response_language: "zh",
    });
  });

  it("sends ready live connectors for broad prompts without an inferred route", async () => {
    mockStreamChatQuery.mockRejectedValue(new Error("stream route unavailable"));
    mockPostChatQuery.mockResolvedValue({
      ok: true,
      body: makeChatResult({ connector: "multiple", provider: "codex" }),
    });
    const state = makeState();

    await runChatQuery({
      text: "kkg payment 決済GW",
      workspacePath: "workspace",
      readySources: [
        { connector: "jira", uri: "jira", status: "ready" },
        { connector: "github", uri: "github", status: "ready" },
        { connector: "filesystem", uri: "docs", status: "ready" },
      ],
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockStreamChatQuery.mock.calls[0][0]).toMatchObject({
      connectors: ["jira", "github"],
    });
    expect(mockPostChatQuery.mock.calls[0][0]).toMatchObject({
      connectors: ["jira", "github"],
    });
  });

  it("streams Codex progress and uses the streamed result", async () => {
    mockStreamChatQuery.mockImplementation(async (_body, handlers) => {
      handlers.onLog("› Live Codex: GitHub plugin lookup");
      handlers.onLog("• Starting Codex CLI exec.");
      handlers.onStatus(2);
      handlers.onAnswer({
        intent: "artifacts",
        workspace_id: "workspace",
        workspace_path: "workspace",
        connector: "github",
        source_uri: "sx-tane/context-os",
        provider: "codex",
        answer: "Live answer",
        summary: "Live summary",
        artifact_count: 0,
        artifacts: [],
        evidence_save_status: "saving",
      });
      handlers.onLog("• Saving live answer evidence to Local DB...");
      handlers.onResult({
        intent: "artifacts",
        workspace_id: "workspace",
        workspace_path: "workspace",
        connector: "github",
        source_uri: "sx-tane/context-os",
        provider: "codex",
        answer: "Live answer",
        summary: "Live summary",
        artifact_count: 0,
        artifacts: [],
        evidence_save_status: "saved",
        evidence_event_count: 2,
        evidence_graph_status: "updated",
      });
    });
    const state = makeState();

    await runChatQuery({
      text: "help me check sx-tane/context-os",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockPostChatQuery).not.toHaveBeenCalled();
    expect(mockStreamChatQuery.mock.calls[0][0]).toMatchObject({
      connector: "github",
      source_uri: "sx-tane/context-os",
    });
    expect(state.addMessage.mock.calls[0][0].text).toBe("");
    expect(state.addMessage.mock.calls[0][0].stream.lines.join("\n")).toContain("GitHub plugin lookup");
    expect(state.replacements[0].stream?.latestLine).toContain("› Live Codex");
    expect(state.replacements.at(-2)?.stream?.latestLine).toContain("• Saving live answer evidence to Local DB...");
    expect(state.replacements.at(-2)?.text).toContain("Live answer");
    expect(state.replacements.at(-2)?.stream?.status).toBe("running");
    expect(state.lastChatResult?.answer).toBe("Live answer");
    expect(state.lastChatResult?.evidence_save_status).toBe("saved");
    expect(state.replacements.at(-1)?.text).toBe("Live answer");
    expect(state.replacements.at(-1)?.stream?.status).toBe("complete");
    expect(state.replacements.at(-1)?.stream?.summary).toBe("Local DB: saved 2 artifacts; graph updated");
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });

  it("sends inferred Jira issue route fields for streamed and fallback queries", async () => {
    mockStreamChatQuery.mockRejectedValue(new Error("stream route unavailable"));
    mockPostChatQuery.mockResolvedValue({
      ok: true,
      body: makeChatResult({
        connector: "jira",
        source_uri: "BKGDEV-8466",
        provider: "codex",
        answer: "Jira answer",
        summary: "Jira answer",
      }),
    });
    const state = makeState();

    await runChatQuery({
      text: "BKGDEV-8466 check this",
      workspacePath: "workspace",
      readySources: [
        { connector: "jira", uri: "jira", status: "ready" },
        { connector: "github", uri: "github", status: "ready" },
      ],
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockStreamChatQuery.mock.calls[0][0]).toMatchObject({
      connector: "jira",
      source_uri: "BKGDEV-8466",
    });
    expect(mockStreamChatQuery.mock.calls[0][0].connectors).toBeUndefined();
    expect(mockPostChatQuery.mock.calls[0][0]).toMatchObject({
      connector: "jira",
      source_uri: "BKGDEV-8466",
    });
    expect(mockPostChatQuery.mock.calls[0][0].connectors).toBeUndefined();
    expect(state.lastChatResult?.source_uri).toBe("BKGDEV-8466");
  });

  it("sends Google Drive connector scope for spreadsheet filename prompts with Jira and Slack words", async () => {
    mockStreamChatQuery.mockRejectedValue(new Error("stream route unavailable"));
    mockPostChatQuery.mockResolvedValue({
      ok: true,
      body: makeChatResult({
        connector: "googledrive",
        source_uri: "googledrive",
        provider: "codex",
      }),
    });
    const state = makeState();

    await runChatQuery({
      text: "BKGDEV-8096_帳票項目のマッピング確認.xlsx のJiraとSlackを確認して",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockStreamChatQuery.mock.calls[0][0]).toMatchObject({
      connector: "googledrive",
    });
    expect(mockStreamChatQuery.mock.calls[0][0].source_uri).toBeUndefined();
    expect(mockPostChatQuery.mock.calls[0][0]).toMatchObject({
      connector: "googledrive",
    });
    expect(mockPostChatQuery.mock.calls[0][0].source_uri).toBeUndefined();
  });

  it("keeps the answer when the stream fails after an early answer", async () => {
    mockStreamChatQuery.mockImplementation(async (_body, handlers) => {
      handlers.onLog("› Live Codex: Jira plugin lookup");
      handlers.onAnswer({
        intent: "artifacts",
        workspace_id: "workspace",
        workspace_path: "workspace",
        connector: "jira",
        source_uri: "BKGDEV-8466",
        provider: "codex",
        answer: "BKGDEV-8466 is done.",
        summary: "BKGDEV-8466 is done.",
        artifact_count: 0,
        artifacts: [],
        evidence_save_status: "saving",
      });
      throw new Error("network error");
    });
    const state = makeState();

    await runChatQuery({
      text: "BKGDEV-8466 check this JIRA",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockPostChatQuery).not.toHaveBeenCalled();
    expect(state.replacements.at(-1)?.text).toBe("BKGDEV-8466 is done.");
    expect(state.replacements.at(-1)?.stream?.status).toBe("complete");
    expect(state.replacements.at(-1)?.stream?.latestLine).toContain("Local DB save failed");
    expect(state.replacements.at(-1)?.stream?.summary).toContain("Local DB: save failed");
    expect(state.lastChatResult?.evidence_save_status).toBe("error");
  });

  it("keeps streamed Codex transcript when live lookup fails", async () => {
    mockStreamChatQuery.mockImplementation(async (_body, handlers) => {
      handlers.onLog("› Live Codex: GitHub plugin lookup");
      handlers.onLog("• Starting Codex CLI exec.");
      handlers.onError("codex live chat timed out after 5m0s");
    });
    const state = makeState();

    await runChatQuery({
      text: "help me check sx-tane/context-os",
      workspacePath: "workspace",
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastChatResult: state.setLastChatResult,
      setActivityArtifacts: state.setActivityArtifacts,
      refreshWorkspace: state.refreshWorkspace,
    });

    expect(mockPostChatQuery).not.toHaveBeenCalled();
    expect(state.replacements.at(-1)?.text).toBe("");
    expect(state.replacements.at(-1)?.loading).toBe(false);
    expect(state.replacements.at(-1)?.stream?.status).toBe("error");
    expect(state.replacements.at(-1)?.stream?.lines.join("\n")).toContain("› Live Codex");
    expect(state.replacements.at(-1)?.stream?.latestLine).toContain("Live Codex lookup failed");
    expect(state.replacements.at(-1)?.stream?.latestLine).toContain("timed out after 5m0s");
  });
});

function makeState(): ChatControllerTestState {
  const busyCalls: boolean[] = [];
  let lastChatResult: ChatQueryResult | null = null;
  const replacements: ChatMessageSnapshot[] = [];
  const state: ChatControllerTestState = {
    busyCalls,
    get lastChatResult() {
      return lastChatResult;
    },
    set lastChatResult(value) {
      lastChatResult = value;
    },
    replacements,
    addMessage: jest.fn(),
    replaceMessage: jest.fn((_id: string, message: ChatMessageSnapshot) => {
      replacements.push(message);
    }),
    setBusy: jest.fn((value: boolean) => busyCalls.push(value)),
    setLastChatResult: jest.fn((result: ChatQueryResult | null) => {
      lastChatResult = result;
    }),
    setActivityArtifacts: jest.fn(),
    refreshWorkspace: jest.fn<Promise<void>, []>().mockResolvedValue(),
  };
  return state;
}

function makeChatResult(overrides: Partial<ChatQueryResult> = {}): ChatQueryResult {
  return {
    intent: "status",
    workspace_id: "workspace",
    workspace_path: "workspace",
    provider: "local",
    answer: "Answer",
    summary: "Summary",
    artifact_count: 0,
    artifacts: [],
    ...overrides,
  };
}
