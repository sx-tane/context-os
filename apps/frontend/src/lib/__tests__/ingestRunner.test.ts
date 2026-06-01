import { runConnectorIngest } from "../ingestRunner";

jest.mock("$lib/api");

import { postIngest, streamCodexIngest } from "$lib/api";

const mockPostIngest = postIngest as jest.Mock;
const mockStreamCodexIngest = streamCodexIngest as jest.Mock;

// makeSetters builds fresh no-op reactive setters for each test.
function makeSetters() {
  return {
    setLoading: jest.fn<void, [boolean]>(),
    setError: jest.fn<void, [string]>(),
    setResult: jest.fn(),
    setLiveLog: jest.fn(),
    setElapsed: jest.fn(),
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

// ---- token provider ----

describe("runConnectorIngest — token provider", () => {
  it("calls postIngest and forwards the result to setResult", async () => {
    const body = { connector: "github", events: [] };
    mockPostIngest.mockResolvedValue({ ok: true, status: 200, body });
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      token: "ghp_test",
      ...s,
    });

    expect(mockPostIngest).toHaveBeenCalledWith(
      "github",
      expect.objectContaining({
        uri: "github://owner/repo",
        token: "ghp_test",
      }),
      expect.any(Object),
    );
    expect(s.setResult).toHaveBeenCalledWith(body);
    // setError is called with "" to clear previous state — assert no non-empty error was set
    const errorCalls = s.setError.mock.calls.filter(
      ([v]: [string]) => v !== "",
    );
    expect(errorCalls).toHaveLength(0);
  });

  it("calls setError with message when postIngest returns ok:false", async () => {
    mockPostIngest.mockResolvedValue({
      ok: false,
      status: 422,
      body: { message: "invalid URI" },
    });
    const s = makeSetters();

    await runConnectorIngest({
      connector: "slack",
      uri: "bad",
      provider: "token",
      ...s,
    });

    expect(s.setError).toHaveBeenCalledWith("invalid URI");
    expect(s.setResult).not.toHaveBeenCalledWith(expect.anything());
  });

  it("sets loading true at start and false in finally", async () => {
    mockPostIngest.mockResolvedValue({ ok: true, status: 200, body: {} });
    const s = makeSetters();
    const calls: boolean[] = [];
    s.setLoading.mockImplementation((v) => calls.push(v));

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      ...s,
    });

    expect(calls[0]).toBe(true);
    expect(calls[calls.length - 1]).toBe(false);
  });

  it("strips empty-string metadata values before sending", async () => {
    mockPostIngest.mockResolvedValue({ ok: true, status: 200, body: {} });
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      metadata: { keep: "value", drop: "  " },
      ...s,
    });

    const sentBody = mockPostIngest.mock.calls[0][1];
    expect(sentBody.metadata).toEqual({ keep: "value" });
    expect(sentBody.metadata).not.toHaveProperty("drop");
  });

  it("swallows AbortError without calling setError", async () => {
    const abortErr = new DOMException("aborted", "AbortError");
    mockPostIngest.mockRejectedValue(abortErr);
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      ...s,
    });

    // setError is called with "" to clear previous state — assert no non-empty error was set
    const errorCalls = s.setError.mock.calls.filter(
      ([v]: [string]) => v !== "",
    );
    expect(errorCalls).toHaveLength(0);
  });

  it("calls setError for non-abort errors", async () => {
    mockPostIngest.mockRejectedValue(new Error("timeout"));
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      ...s,
    });

    expect(s.setError).toHaveBeenCalledWith("timeout");
  });
});

// ---- codex provider ----

describe("runConnectorIngest — codex provider", () => {
  it("calls streamCodexIngest for a supported connector", async () => {
    mockStreamCodexIngest.mockResolvedValue(undefined);
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "codex",
      ...s,
    });

    expect(mockStreamCodexIngest).toHaveBeenCalledWith(
      "github",
      expect.objectContaining({
        uri: "github://owner/repo",
        provider: "codex",
      }),
      expect.any(Object),
      expect.any(Object),
    );
    expect(mockPostIngest).not.toHaveBeenCalled();
  });

  it("calls setError without calling the API when connector is 'filesystem'", async () => {
    const s = makeSetters();

    await runConnectorIngest({
      connector: "filesystem",
      uri: "docs/",
      provider: "codex",
      ...s,
    });

    expect(s.setError).toHaveBeenCalledWith(
      expect.stringContaining("not supported"),
    );
    expect(mockStreamCodexIngest).not.toHaveBeenCalled();
  });
});

// ---- isCurrent guard ----

describe("runConnectorIngest — isCurrent guard", () => {
  it("does not call setters when isCurrent returns false", async () => {
    mockPostIngest.mockResolvedValue({ ok: true, status: 200, body: {} });
    const s = makeSetters();

    await runConnectorIngest({
      connector: "github",
      uri: "github://owner/repo",
      provider: "token",
      isCurrent: () => false,
      ...s,
    });

    expect(s.setLoading).not.toHaveBeenCalled();
    expect(s.setResult).not.toHaveBeenCalled();
  });
});
