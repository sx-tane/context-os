jest.mock("$lib/api");
jest.mock("$lib/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  assistantMsg,
  classifyChatCommand,
  localDateString,
  runChatQuery,
  userMsg,
} from "../chatController";

import { postChatQuery } from "$lib/api";

const mockPostChatQuery = postChatQuery as jest.Mock;

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

describe("runChatQuery", () => {
  it("posts source queries and updates chat state on success", async () => {
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

    expect(mockPostChatQuery).toHaveBeenCalledWith(
      expect.objectContaining({ workspace_id: "workspace", message: "status" }),
    );
    expect(state.busyCalls).toEqual([true, false]);
    expect(state.lastChatResult?.answer).toBe("Answer");
    expect(state.replacements.at(-1)?.text).toBe("Answer");
    expect(state.refreshWorkspace).toHaveBeenCalled();
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
