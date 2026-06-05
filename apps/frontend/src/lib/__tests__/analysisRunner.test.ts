jest.mock("$lib/api");
jest.mock("$lib/workspace/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  analysisTimeoutMs,
  analysisProvider,
  buildAnalysisProgress,
  runAnalysis,
} from "../findings/analysisRunner";

import { postFindings } from "$lib/api";
import type {
  Artifact,
  ChatQueryResult,
  ConnectorKnowledge,
  FindingsResult,
} from "../types";

const mockPostFindings = postFindings as jest.Mock;

type AnalysisRunnerTestState = {
  busyCalls: boolean[];
  lastFindings: FindingsResult | null;
  replacements: Array<{ text: string; card?: unknown }>;
  addMessage: jest.Mock;
  replaceMessage: jest.Mock;
  setBusy: jest.Mock;
  setLastFindings: jest.Mock;
  setLastAnalysisAt: jest.Mock;
  openSources: jest.Mock;
  refreshWorkspace: jest.Mock<Promise<void>, []>;
};

beforeEach(() => {
  mockPostFindings.mockReset();
});

describe("analysisProvider", () => {
  it("uses codex for plugin-backed connectors and token for filesystem", () => {
    expect(analysisProvider("github")).toBe("codex");
    expect(analysisProvider("slack")).toBe("codex");
    expect(analysisProvider("filesystem")).toBe("token");
  });
});

describe("analysisTimeoutMs", () => {
  it("allows slow Codex-backed sources to run as long as the API findings timeout", () => {
    expect(analysisTimeoutMs("codex")).toBe(300000);
    expect(analysisTimeoutMs("token")).toBe(90000);
    expect(analysisTimeoutMs("codex", 1000)).toBe(1000);
  });
});

describe("buildAnalysisProgress", () => {
  it("summarizes queued, running, done, and failed source statuses", () => {
    const message = buildAnalysisProgress([
      { connector: "github", uri: "repo", status: "done", detail: "2 events, 1 findings" },
      { connector: "slack", uri: "channel", status: "failed", detail: "unauthorized" },
      { connector: "jira", uri: "DEMO", status: "running" },
      { connector: "filesystem", uri: "/tmp", status: "queued" },
    ]);

    expect(message).toContain("1/4 complete, 1 failed");
    expect(message).toContain("1. github:repo - done (2 events, 1 findings)");
    expect(message).toContain("2. slack:channel - failed: unauthorized");
    expect(message).toContain("3. jira:DEMO - running");
    expect(message).toContain("4. filesystem:/tmp - queued");
  });
});

describe("runAnalysis", () => {
  it("aggregates successful findings and reports per-source failures", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings
      .mockResolvedValueOnce({
        ok: true,
        body: {
          connector: "github",
          uri: "repo",
          mismatch_count: 1,
          review_candidate_count: 4,
          event_count: 2,
          entity_count: 3,
          mismatches: [{ id: "m1", severity: "high", summary: "API field drift" }],
          review_candidates: [{ id: "dependency_risk:r1", type: "dependency_review" }],
          mismatch_ids: ["m1"],
        },
      })
      .mockResolvedValueOnce({
        ok: false,
        body: { message: "unauthorized" },
      });
    const state = makeState();

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [
          { connector: "github", uri: "repo", status: "ready" },
          { connector: "slack", uri: "channel", status: "ready" },
        ],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(mockPostFindings).toHaveBeenCalledTimes(2);
    expect(state.busyCalls).toEqual([true, false]);
    expect(state.lastFindings?.mismatch_count).toBe(1);
    const finalMessage = state.replacements.at(-1)!;
    expect(finalMessage.text).toContain("Found 1 actionable finding");
    expect(finalMessage.text).toContain("4 review candidates");
    expect(finalMessage.text).toContain("**Top issues**");
    expect(finalMessage.text).toContain("1. API field drift");
    expect(finalMessage.text).not.toContain("Findings preview is attached");
    expect(finalMessage.text).toContain("Failed:");
    expect(finalMessage.text).toContain("slack:channel - unauthorized");
    expect(finalMessage.card).toMatchObject({ kind: "findings" });
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });

  it("removes the parent abort listener after a source completes", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings.mockResolvedValueOnce({
      ok: true,
      body: {
        connector: "github",
        uri: "repo",
        mismatch_count: 0,
        event_count: 1,
        entity_count: 1,
        mismatches: [],
      },
    });
    const state = makeState();
    const controller = new AbortController();
    const add = jest.spyOn(controller.signal, "addEventListener");
    const remove = jest.spyOn(controller.signal, "removeEventListener");

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [{ connector: "github", uri: "repo", status: "ready" }],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        signal: controller.signal,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(add).toHaveBeenCalledWith("abort", expect.any(Function), { once: true });
    expect(remove).toHaveBeenCalledWith("abort", expect.any(Function));
  });

  it("skips broad connector scopes before calling findings analysis", async () => {
    const state = makeState();

    await runAnalysis({
      workspacePath: "workspace",
      readySources: [
        { connector: "github", uri: "github", status: "ready" },
        { connector: "slack", uri: "slack", status: "ready" },
      ],
      addMessage: state.addMessage,
      replaceMessage: state.replaceMessage,
      setBusy: state.setBusy,
      setLastFindings: state.setLastFindings,
      setLastAnalysisAt: state.setLastAnalysisAt,
      openSources: state.openSources,
      refreshWorkspace: state.refreshWorkspace,
      timeoutMs: 1000,
    });

    expect(mockPostFindings).not.toHaveBeenCalled();
    expect(state.addMessage.mock.calls.at(-1)?.[0].text).toContain("No concrete source was ready");
    expect(state.addMessage.mock.calls.at(-1)?.[0].text).toContain("Skipped chat-only scopes");
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });

  it("runs concrete sources and reports skipped broad connector scopes separately", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings.mockResolvedValueOnce({
      ok: true,
      body: {
        connector: "github",
        uri: "owner/repo",
        mismatch_count: 0,
        event_count: 2,
        entity_count: 3,
        mismatches: [],
      },
    });
    const state = makeState();

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [
          { connector: "github", uri: "github", status: "ready" },
          { connector: "github", uri: "owner/repo", status: "ready" },
        ],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(mockPostFindings).toHaveBeenCalledTimes(1);
    expect(mockPostFindings.mock.calls[0][0]).toMatchObject({
      connector: "github",
      uri: "owner/repo",
    });
    expect(state.replacements.at(-1)?.text).toContain("1/1 concrete source");
    expect(state.replacements.at(-1)?.text).toContain("Skipped chat-only scopes");
    expect(state.replacements.at(-1)?.text).not.toContain("Failed:");
  });

  it("runs chat-derived concrete sources instead of skipping broad connector scopes", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings.mockResolvedValueOnce({
      ok: true,
      body: {
        connector: "jira",
        uri: "BKGDEV-8551",
        mismatch_count: 0,
        event_count: 1,
        entity_count: 2,
        mismatches: [],
      },
    });
    const state = makeState();

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [
          { connector: "jira", uri: "jira", status: "ready" },
          { connector: "slack", uri: "slack", status: "ready" },
        ],
        lastChatResult: chatResult({
          answer_sections: [
            {
              source_label: "Jira issue BKGDEV-8551",
              connector: "jira",
              source_uri: "BKGDEV-8551",
            },
          ],
        }),
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(mockPostFindings).toHaveBeenCalledTimes(1);
    expect(mockPostFindings.mock.calls[0][0]).toMatchObject({
      connector: "jira",
      uri: "BKGDEV-8551",
    });
    expect(state.replacements.at(-1)?.text).toContain("1/1 concrete source");
    expect(state.replacements.at(-1)?.text).toContain("Skipped chat-only scopes");
  });

  it("runs activity-derived live evidence even when no Sources row is ready", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings.mockResolvedValueOnce({
      ok: true,
      body: {
        connector: "github",
        uri: "https://github.com/context-os/app/pull/43",
        mismatch_count: 0,
        event_count: 1,
        entity_count: 1,
        mismatches: [],
      },
    });
    const state = makeState();

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [],
        recentArtifacts: [
          artifact({
            connector: "github",
            source_uri: "https://github.com/context-os/app/pull/43",
            metadata: {
              evidence_kind: "live_chat_answer",
            },
          }),
        ],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(state.openSources).not.toHaveBeenCalled();
    expect(mockPostFindings).toHaveBeenCalledTimes(1);
    expect(mockPostFindings.mock.calls[0][0]).toMatchObject({
      connector: "github",
      uri: "https://github.com/context-os/app/pull/43",
    });
  });

  it("uses basket sources only when the analysis basket has items", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    mockPostFindings.mockResolvedValueOnce({
      ok: true,
      body: {
        connector: "slack",
        uri: "#release",
        mismatch_count: 0,
        event_count: 1,
        entity_count: 1,
        mismatches: [],
      },
    });
    const state = makeState();

    try {
      await runAnalysis({
        workspacePath: "workspace",
        readySources: [
          { connector: "github", uri: "owner/repo", status: "ready" },
        ],
        basketItems: [
          {
            id: "slack:#release",
            connector: "slack",
            uri: "#release",
            label: "Release",
            origin: "activity",
            addedAt: "2026-06-04T00:00:00.000Z",
          },
        ],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        timeoutMs: 1000,
      });
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(mockPostFindings).toHaveBeenCalledTimes(1);
    expect(mockPostFindings.mock.calls[0][0]).toMatchObject({
      connector: "slack",
      uri: "#release",
    });
  });

  it("cancels the running source and leaves queued sources unrequested", async () => {
    const originalWindow = (global as unknown as { window?: unknown }).window;
    (global as unknown as { window: Pick<typeof window, "setTimeout" | "clearTimeout"> }).window = {
      setTimeout,
      clearTimeout,
    };
    const controller = new AbortController();
    mockPostFindings.mockImplementation((_body, options?: { signal?: AbortSignal }) =>
      new Promise((_, reject) => {
        options?.signal?.addEventListener("abort", () => {
          reject(new DOMException("aborted", "AbortError"));
        });
      }),
    );
    const state = makeState();

    try {
      const run = runAnalysis({
        workspacePath: "workspace",
        readySources: [
          { connector: "jira", uri: "BKGDEV-8528", status: "ready" },
          { connector: "github", uri: "owner/repo", status: "ready" },
        ],
        addMessage: state.addMessage,
        replaceMessage: state.replaceMessage,
        setBusy: state.setBusy,
        setLastFindings: state.setLastFindings,
        setLastAnalysisAt: state.setLastAnalysisAt,
        openSources: state.openSources,
        refreshWorkspace: state.refreshWorkspace,
        signal: controller.signal,
      });
      await Promise.resolve();
      controller.abort();
      await run;
    } finally {
      (global as unknown as { window?: unknown }).window = originalWindow;
    }

    expect(mockPostFindings).toHaveBeenCalledTimes(1);
    expect(state.replacements.at(-1)?.text).toContain("Analysis canceled.");
    expect(state.replacements.at(-1)?.text).toContain("1. jira:BKGDEV-8528 - canceled");
    expect(state.replacements.at(-1)?.text).toContain("2. github:owner/repo - canceled");
    expect(state.busyCalls).toEqual([true, false]);
    expect(state.refreshWorkspace).toHaveBeenCalled();
  });
});

function makeState(): AnalysisRunnerTestState {
  const busyCalls: boolean[] = [];
  let lastFindings: FindingsResult | null = null;
  const replacements: Array<{ text: string }> = [];
  const state: AnalysisRunnerTestState = {
    busyCalls,
    get lastFindings() {
      return lastFindings;
    },
    set lastFindings(value) {
      lastFindings = value;
    },
    replacements,
    addMessage: jest.fn(),
    replaceMessage: jest.fn((_id: string, message: { text: string; card?: unknown }) => {
      replacements.push(message);
    }),
    setBusy: jest.fn((value: boolean) => busyCalls.push(value)),
    setLastFindings: jest.fn((result: FindingsResult | null) => {
      lastFindings = result;
    }),
    setLastAnalysisAt: jest.fn(),
    openSources: jest.fn(),
    refreshWorkspace: jest.fn<Promise<void>, []>().mockResolvedValue(),
  };
  return state;
}

function chatResult(overrides: Partial<ChatQueryResult>): ChatQueryResult {
  return {
    intent: "artifacts",
    workspace_id: "workspace",
    workspace_path: "workspace",
    provider: "codex",
    answer: "Answer",
    summary: "Summary",
    artifact_count: 0,
    artifacts: [],
    ...overrides,
  };
}

function artifact(overrides: Partial<Artifact>): Artifact {
  return {
    id: "artifact",
    workspace_id: "workspace",
    connector: "github",
    source_uri: "owner/repo",
    event_type: "document.ingested",
    title: "Evidence",
    body: "Evidence body",
    preview: "Evidence preview",
    content_hash: "hash",
    schema_version: "1",
    ingested_at: "2026-06-04T00:00:00.000Z",
    ...overrides,
  };
}
