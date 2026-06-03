jest.mock("$lib/api");
jest.mock("$lib/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  assistantMsg,
  buildChatLoadingText,
  classifyChatCommand,
  localDateString,
  runChatQuery,
  userMsg,
} from "../chatController";

import { postChatQuery, streamChatQuery } from "$lib/api";

const mockPostChatQuery = postChatQuery as jest.Mock;
const mockStreamChatQuery = streamChatQuery as jest.Mock;

beforeEach(() => {
  mockPostChatQuery.mockReset();
  mockStreamChatQuery.mockReset();
});

type ChatControllerTestState = {
  busyCalls: boolean[];
  lastChatResult: { answer: string } | null;
  replacements: Array<{ text: string }>;
  addMessage: jest.Mock;
  replaceMessage: jest.Mock;
  setBusy: jest.Mock;
  setLastChatResult: jest.Mock;
  setActivityArtifacts: jest.Mock;
  refreshWorkspace: jest.Mock<Promise<void>, []>;
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

describe("buildChatLoadingText", () => {
  it("shows live Codex route for owner/repo prompts", () => {
    const text = buildChatLoadingText("help me check sx-tane/context-os");

    expect(text).toContain("Live Codex");
    expect(text).toContain("GitHub plugin lookup: sx-tane/context-os");
    expect(text).toContain("Codex CLI");
    expect(text).toContain("Local DB");
  });

  it("shows local DB route for local file prompts", () => {
    const text = buildChatLoadingText("summarize local file docs/plan.md");

    expect(text).toContain("Local DB");
    expect(text).not.toContain("Live Codex");
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
      expect.objectContaining({ workspace_id: "workspace", message: "status" }),
      expect.objectContaining({
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
    expect(state.addMessage.mock.calls[0][0].text).toContain("Local DB");
    expect(state.replacements[0].text).toContain("Streaming unavailable");
    expect(state.replacements[0].text).toContain("standard chat query");
    expect(state.lastChatResult?.answer).toBe("Answer");
    expect(state.replacements.at(-1)?.text).toBe("Answer");
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });

  it("streams Codex progress and uses the streamed result", async () => {
    mockStreamChatQuery.mockImplementation(async (_body, handlers) => {
      handlers.onLog("› Live Codex: GitHub plugin lookup");
      handlers.onLog("• Starting Codex CLI exec.");
      handlers.onStatus(2);
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
    expect(state.addMessage.mock.calls[0][0].text).toContain("GitHub plugin lookup");
    expect(state.replacements[0].text).toContain("› Live Codex");
    expect(state.replacements.at(-2)?.text).toContain("• Codex still running... 2s");
    expect(state.lastChatResult?.answer).toBe("Live answer");
    expect(state.replacements.at(-1)?.text).toBe("Live answer");
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
    expect(state.replacements.at(-1)?.text).toContain("› Live Codex");
    expect(state.replacements.at(-1)?.text).toContain("Live Codex lookup failed");
    expect(state.replacements.at(-1)?.text).toContain("timed out after 5m0s");
  });
});

function makeState(): ChatControllerTestState {
  const busyCalls: boolean[] = [];
  let lastChatResult: { answer: string } | null = null;
  const replacements: Array<{ text: string }> = [];
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
    replaceMessage: jest.fn((_id: string, message: { text: string }) => {
      replacements.push(message);
    }),
    setBusy: jest.fn((value: boolean) => busyCalls.push(value)),
    setLastChatResult: jest.fn((result: { answer: string } | null) => {
      lastChatResult = result;
    }),
    setActivityArtifacts: jest.fn(),
    refreshWorkspace: jest.fn<Promise<void>, []>().mockResolvedValue(),
  };
  return state;
}
