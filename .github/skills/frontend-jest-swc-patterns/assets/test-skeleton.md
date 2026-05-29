# Frontend Test Skeleton

Copy this as the starting point for a new `*.test.ts` file under `src/lib/__tests__/`.
Replace all `<placeholder>` values. Delete sections that do not apply.

---

## Pure utility with `global.fetch` — e.g. `api.ts`

```typescript
import { <functionName> } from "../<module>";

// makeResponse builds a minimal fetch Response mock for readJSON (calls res.text()).
function makeResponse(body: unknown, ok: boolean, status = 200): Response {
  return {
    ok,
    status,
    text: jest.fn().mockResolvedValue(
      typeof body === "string" ? body : JSON.stringify(body),
    ),
  } as unknown as Response;
}

const fetchMock = jest.fn<Promise<Response>, [RequestInfo | URL, RequestInit?]>();
(global as unknown as Record<string, unknown>).fetch = fetchMock;

beforeEach(() => {
  fetchMock.mockReset();
});

// ---- <groupName> ----

describe("<functionName>", () => {
  it("<present-tense description of behaviour and expected outcome>", async () => {
    fetchMock.mockResolvedValue(makeResponse({ key: "value" }, true, 200));

    const result = await <functionName>(<args>);

    expect(result).toEqual(<expected>);
  });

  it("returns <fallback> when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));

    const result = await <functionName>(<args>);

    expect(result).toBe(<fallback>);
  });
});
```

---

## Orchestration module with mocked `$lib/` dependencies — e.g. `ingestRunner.ts`

```typescript
// jest.mock MUST appear before the subject import for SWC hoisting to apply.
jest.mock("$lib/api");

import { <functionName> } from "../<module>";
import { <mockTarget> } from "$lib/api";

const mock<MockTarget> = <mockTarget> as jest.Mock;

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

// ---- happy path ----

describe("<functionName> — <scenario group>", () => {
  it("<present-tense description of behaviour and expected outcome>", async () => {
    mock<MockTarget>.mockResolvedValue(<returnValue>);
    const s = makeSetters();

    await <functionName>({ <args>, ...s });

    expect(mock<MockTarget>).toHaveBeenCalledWith(<expectedArgs>);
    expect(s.setResult).toHaveBeenCalledWith(<expectedResult>);
    // Verify no non-empty error was set (empty-string clear is expected)
    const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
    expect(errorCalls).toHaveLength(0);
  });
});

// ---- lifecycle ----

describe("<functionName> — lifecycle", () => {
  it("sets loading true at start and false in finally", async () => {
    mock<MockTarget>.mockResolvedValue(<returnValue>);
    const s = makeSetters();
    const calls: boolean[] = [];
    s.setLoading.mockImplementation((v) => calls.push(v));

    await <functionName>({ <args>, ...s });

    expect(calls[0]).toBe(true);
    expect(calls[calls.length - 1]).toBe(false);
  });
});

// ---- abort ----

describe("<functionName> — abort handling", () => {
  it("swallows AbortError without setting a non-empty error", async () => {
    mock<MockTarget>.mockRejectedValue(new DOMException("aborted", "AbortError"));
    const s = makeSetters();

    await <functionName>({ <args>, ...s });

    const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
    expect(errorCalls).toHaveLength(0);
  });
});

// ---- isCurrent guard ----

describe("<functionName> — isCurrent guard", () => {
  it("does not call setters when isCurrent returns false", async () => {
    mock<MockTarget>.mockResolvedValue(<returnValue>);
    const s = makeSetters();

    await <functionName>({ <args>, isCurrent: () => false, ...s });

    expect(s.setLoading).not.toHaveBeenCalled();
    expect(s.setResult).not.toHaveBeenCalled();
  });
});
```

---

## Module with function-updater setters — e.g. `reauthRunner.ts`

```typescript
jest.mock("$lib/api");

import { <functionName> } from "../<module>";
import { <mockTarget> } from "$lib/api";

const mock<MockTarget> = <mockTarget> as jest.Mock;

function makeOptions() {
  return {
    <param>: "<value>",
    refresh<Resource>: jest.fn<Promise<void>, []>().mockResolvedValue(undefined),
    set<Prop>: jest.fn<void, [string]>(),
    setLog: jest.fn(),
    setRunning: jest.fn<void, [boolean]>(),
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe("<functionName> — happy path", () => {
  it("calls <mockTarget> with the expected argument", async () => {
    mock<MockTarget>.mockResolvedValue(undefined);
    const opts = makeOptions();

    await <functionName>(opts);

    expect(mock<MockTarget>).toHaveBeenCalledWith(
      "<expectedArg>",
      expect.any(Function),
      expect.any(Object),
    );
  });

  it("calls refresh<Resource> in finally even when the stream throws", async () => {
    mock<MockTarget>.mockRejectedValue(new Error("fail"));
    const opts = makeOptions();

    await <functionName>(opts);

    expect(opts.refresh<Resource>).toHaveBeenCalledTimes(1);
  });

  it("appends non-abort errors to the log via function updater", async () => {
    mock<MockTarget>.mockRejectedValue(new Error("<error message>"));
    const opts = makeOptions();
    const logged: string[] = [];
    opts.setLog.mockImplementation((v: string | ((s: string) => string)) => {
      // Error appenders use (current) => current + msg + "\n"
      if (typeof v === "function") logged.push(v(""));
      else logged.push(v);
    });

    await <functionName>(opts);

    expect(logged.some((l) => l.includes("<error message>"))).toBe(true);
  });
});
```
