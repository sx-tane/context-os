import { runCodexReauth } from "../ingest/reauthRunner";

jest.mock("$lib/api");

import { streamCodexReauth } from "$lib/api";

const mockStreamCodexReauth = streamCodexReauth as jest.Mock;

// makeOptions builds a fresh options object with jest mock setters.
function makeOptions(
  overrides: {
    isCurrent?: () => boolean;
    signal?: AbortSignal;
  } = {},
) {
  return {
    plugin: "github",
    refreshCodexStatus: jest
      .fn<Promise<void>, []>()
      .mockResolvedValue(undefined),
    setPlugin: jest.fn<void, [string]>(),
    setLog: jest.fn(),
    setRunning: jest.fn<void, [boolean]>(),
    ...overrides,
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

// ---- happy path ----

describe("runCodexReauth — happy path", () => {
  it("calls streamCodexReauth with the plugin name", async () => {
    mockStreamCodexReauth.mockResolvedValue(undefined);
    const opts = makeOptions();

    await runCodexReauth(opts);

    expect(mockStreamCodexReauth).toHaveBeenCalledWith(
      "github",
      expect.any(Function),
      expect.any(Object),
    );
  });

  it("appends each log line to the accumulated log", async () => {
    mockStreamCodexReauth.mockImplementation(
      async (_plugin: string, onLog: (line: string) => void) => {
        onLog("line one");
        onLog("line two");
      },
    );
    const opts = makeOptions();

    await runCodexReauth(opts);

    // setLog is called once to clear ("") and once per log line from the stream
    const fnUpdaterCalls = opts.setLog.mock.calls.filter(
      ([v]: [unknown]) => typeof v === "function",
    );
    expect(fnUpdaterCalls).toHaveLength(2);
  });

  it("calls refreshCodexStatus after the stream completes", async () => {
    mockStreamCodexReauth.mockResolvedValue(undefined);
    const opts = makeOptions();

    await runCodexReauth(opts);

    expect(opts.refreshCodexStatus).toHaveBeenCalledTimes(1);
  });

  it("sets setRunning true at start and false in finally", async () => {
    mockStreamCodexReauth.mockResolvedValue(undefined);
    const opts = makeOptions();
    const calls: boolean[] = [];
    opts.setRunning.mockImplementation((v) => calls.push(v));

    await runCodexReauth(opts);

    expect(calls[0]).toBe(true);
    expect(calls[calls.length - 1]).toBe(false);
  });

  it("resets setPlugin to empty string in finally", async () => {
    mockStreamCodexReauth.mockResolvedValue(undefined);
    const opts = makeOptions();

    await runCodexReauth(opts);

    const calls = opts.setPlugin.mock.calls.map(([v]: [string]) => v);
    expect(calls[0]).toBe("github");
    expect(calls[calls.length - 1]).toBe("");
  });
});

// ---- error handling ----

describe("runCodexReauth — error handling", () => {
  it("swallows AbortError without appending it to the log", async () => {
    const abortErr = new DOMException("aborted", "AbortError");
    mockStreamCodexReauth.mockRejectedValue(abortErr);
    const opts = makeOptions();

    await runCodexReauth(opts);

    // No log line should contain the abort error
    const logCalls = opts.setLog.mock.calls.filter(([v]: [unknown]) =>
      typeof v === "function" ? false : String(v).includes("AbortError"),
    );
    expect(logCalls).toHaveLength(0);
  });

  it("still calls refreshCodexStatus even when the stream throws", async () => {
    mockStreamCodexReauth.mockRejectedValue(new Error("stream failed"));
    const opts = makeOptions();

    await runCodexReauth(opts);

    expect(opts.refreshCodexStatus).toHaveBeenCalledTimes(1);
  });

  it("appends non-abort errors to the log", async () => {
    mockStreamCodexReauth.mockRejectedValue(new Error("connection refused"));
    const opts = makeOptions();
    const logged: string[] = [];
    opts.setLog.mockImplementation((v: string | ((s: string) => string)) => {
      // error appending uses a function updater: (current) => current + msg + "\n"
      if (typeof v === "function") logged.push(v(""));
      else logged.push(v);
    });

    await runCodexReauth(opts);

    expect(logged.some((l) => l.includes("connection refused"))).toBe(true);
  });
});

// ---- isCurrent guard ----

describe("runCodexReauth — isCurrent guard", () => {
  it("does not call setters when isCurrent returns false", async () => {
    mockStreamCodexReauth.mockResolvedValue(undefined);
    const opts = makeOptions({ isCurrent: () => false });

    await runCodexReauth(opts);

    expect(opts.setPlugin).not.toHaveBeenCalled();
    expect(opts.setRunning).not.toHaveBeenCalled();
    expect(opts.setLog).not.toHaveBeenCalled();
  });
});
