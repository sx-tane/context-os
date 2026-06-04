---
name: frontend-jest-swc-patterns
description: "Apply the canonical ContextOS frontend test pattern when writing or reviewing TypeScript test files under apps/frontend. Use when: adding a new *.test.ts file; reviewing an existing test for style conformance; mocking $lib/ modules; testing reactive setter lifecycles; testing fetch-based API helpers with mocked global fetch; choosing between flat vs describe/it structure."
argument-hint: "Which lib module or utility are you testing?"
user-invocable: true
---

# ContextOS Frontend Jest + SWC Test Patterns

## Outcome

Frontend TypeScript tests are fast, deterministic, mockable, and consistent with the repo's runner conventions.

## Procedure

1. Place the test in `src/lib/__tests__/` with a source-matching `*.test.ts` file name.
2. Put `jest.mock("$lib/…")` before the subject import so SWC hoisting works.
3. Use the relevant skeleton from [Test Skeleton](./assets/test-skeleton.md).
4. Run the [Conformance Checklist](./references/test-checklist.md) and `bun run test` before marking complete.

## Purpose

Every `*.test.ts` file in `apps/frontend/src/lib/__tests__/` must follow one canonical pattern
so tests are readable, mockable, and consistent with the Go test conventions used elsewhere in
this repo.

---

## 1. File Location and Naming

| Subject                     | Test file path                           |
| --------------------------- | ---------------------------------------- |
| `src/lib/api.ts`            | `src/lib/__tests__/api.test.ts`          |
| `src/lib/ingestRunner.ts`   | `src/lib/__tests__/ingestRunner.test.ts` |
| `src/lib/reauthRunner.ts`   | `src/lib/__tests__/reauthRunner.test.ts` |
| Any new `src/lib/<name>.ts` | `src/lib/__tests__/<name>.test.ts`       |

**Rule:** All test files live in `src/lib/__tests__/`. The directory name stays singular and
the file name mirrors the source file exactly, with `.test.ts` appended.

---

## 2. Import Order

Three groups separated by blank lines:

```typescript
// 1. Jest mocks — must come before subject imports so SWC hoisting applies
jest.mock("$lib/api");

// 2. Subject under test
import { runConnectorIngest } from "../ingestRunner";

// 3. Mocked module re-imports for typed access
import { postIngest, streamCodexIngest } from "$lib/api";
```

- `jest.mock()` calls **always appear before the subject import** so SWC hoisting puts them
  at module scope before any `require()`.
- Re-import the mocked module after the subject import and cast to `jest.Mock` or `jest.MockedFunction`.

---

## 3. Mocking `$lib/` Path Aliases

The `moduleNameMapper` in `jest.config.cjs` resolves `$lib/` → `src/lib/`. Use the full
alias path in both the mock call and the import:

```typescript
jest.mock("$lib/api"); // resolves via moduleNameMapper
import { postIngest } from "$lib/api"; // same resolution — gets the mock
const mockPostIngest = postIngest as jest.Mock;
```

Never use relative paths like `"../api"` in `jest.mock()` — keep the mock path and the
source path identical.

---

## 4. Mocking `global.fetch`

All fetch-based tests replace the global directly with a typed jest function:

```typescript
const fetchMock = jest.fn<
  Promise<Response>,
  [RequestInfo | URL, RequestInit?]
>();
(global as unknown as Record<string, unknown>).fetch = fetchMock;

beforeEach(() => {
  fetchMock.mockReset();
});
```

**`makeResponse` helper** — every test file that calls `readJSON` internally needs a response
whose `.text()` resolves to JSON:

```typescript
function makeResponse(body: unknown, ok: boolean, status = 200): Response {
  return {
    ok,
    status,
    text: jest
      .fn()
      .mockResolvedValue(
        typeof body === "string" ? body : JSON.stringify(body),
      ),
  } as unknown as Response;
}
```

For `probeService` (only checks `res.ok`), cast `{ ok: true }` directly — no `text()` needed.

---

## 5. Mocking Svelte Reactive Setters

Modules like `ingestRunner` and `reauthRunner` accept setter callbacks that components pass
from reactive state. Create them with a builder function per test:

```typescript
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
```

**Rules:**

- Call `makeSetters()` inside each test — never share a setter object between tests.
- When a setter accepts a function updater `(current: T) => T`, check for it explicitly:

```typescript
// Error appenders use a function updater — simulate by calling it with the empty state.
opts.setLog.mockImplementation((v: string | ((s: string) => string)) => {
  if (typeof v === "function") logged.push(v(""));
  else logged.push(v);
});
```

---

## 6. Describe / It Structure

Mirror the Go `describe` convention: one outer `describe` per exported function, inner
`describe` blocks per scenario group, `it` for individual cases.

```typescript
describe("runConnectorIngest — token provider", () => {
  it("calls postIngest and forwards the result to setResult", async () => { … });
  it("calls setError with message when postIngest returns ok:false", async () => { … });
  it("swallows AbortError without setting a non-empty error", async () => { … });
});

describe("runConnectorIngest — codex provider", () => {
  it("calls streamCodexIngest for a supported connector", async () => { … });
  it("calls setError without hitting the API when connector is 'filesystem'", async () => { … });
});

describe("runConnectorIngest — isCurrent guard", () => {
  it("does not call setters when isCurrent returns false", async () => { … });
});
```

**Naming rules:**

- Outer: `"<functionName> — <scenario group>"`
- Inner `it`: present-tense, describes the behaviour and expected outcome in one sentence.

---

## 7. Asserting on Lifecycle Setters

Lifecycle setters (`setLoading`, `setRunning`) must be called `true` first and `false` last
(in `finally`). Capture the call sequence:

```typescript
const calls: boolean[] = [];
s.setLoading.mockImplementation((v) => calls.push(v));

await runConnectorIngest({ … });

expect(calls[0]).toBe(true);
expect(calls[calls.length - 1]).toBe(false);
```

---

## 8. Asserting on Clear-then-Set Patterns

Functions that reset state before each run call setters with empty values first (e.g.,
`setError("")`). Never assert `not.toHaveBeenCalledWith(expect.any(String))` — it will
incorrectly fail on the clearing call:

```typescript
// Wrong — fails because setError("") is a legitimate clear
expect(s.setError).not.toHaveBeenCalledWith(expect.any(String));

// Correct — filter out the empty-string clear and assert no non-empty errors remain
const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
expect(errorCalls).toHaveLength(0);
```

---

## 9. AbortError Swallowing

Test that abort errors are silently dropped by triggering the abort path and checking no
error setter was called with a real message:

```typescript
it("swallows AbortError without setting a non-empty error", async () => {
  mockPostIngest.mockRejectedValue(new DOMException("aborted", "AbortError"));
  const s = makeSetters();

  await runConnectorIngest({
    connector: "github",
    uri: "…",
    provider: "token",
    ...s,
  });

  const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
  expect(errorCalls).toHaveLength(0);
});
```

`DOMException` is available as a global in Node 18+ — no import or jsdom required.

---

## 10. `isCurrent` Guard Tests

The guard prevents stale state updates after a component is torn down. Test it by passing
`isCurrent: () => false` and asserting no setter was invoked at all:

```typescript
it("does not call setters when isCurrent returns false", async () => {
  mockPostIngest.mockResolvedValue({ ok: true, status: 200, body: {} });
  const s = makeSetters();

  await runConnectorIngest({
    connector: "github",
    uri: "…",
    provider: "token",
    isCurrent: () => false,
    ...s,
  });

  expect(s.setLoading).not.toHaveBeenCalled();
  expect(s.setResult).not.toHaveBeenCalled();
});
```

---

## 11. Coverage

Run coverage with:

```bash
bun run test:coverage
```

Aim for full branch coverage on pure utility functions (`api.ts`, `ingestRunner.ts`,
`reauthRunner.ts`). Svelte component tests (if added) may have lower coverage targets.

---

## 12. README Alignment

Update frontend README files when the test change changes how the folder is understood:

- `apps/frontend/README.md` — test scripts, Jest/SWC setup, dependencies, or commands.
- `apps/frontend/src/lib/README.md` — exported helper behavior, shared runner contracts, or new test-covered utilities.
- `.github/README.md` — only when the canonical frontend test pattern or customization routing changes.

---

## References

- [Test Skeleton](./assets/test-skeleton.md) — copy-paste starting point for a new `*.test.ts` file.
- [Conformance Checklist](./references/test-checklist.md) — use when reviewing a test file.
- [Run Script](./scripts/run-tests.sh) — convenience wrapper for the Jest suite.
- Real examples in this repo:
  - Fetch mock pattern: [`apps/frontend/src/lib/__tests__/api.test.ts`](../../../../apps/frontend/src/lib/__tests__/api.test.ts)
  - Setter lifecycle + isCurrent: [`apps/frontend/src/lib/__tests__/ingestRunner.test.ts`](../../../../apps/frontend/src/lib/__tests__/ingestRunner.test.ts)
  - Function-updater + finally: [`apps/frontend/src/lib/__tests__/reauthRunner.test.ts`](../../../../apps/frontend/src/lib/__tests__/reauthRunner.test.ts)
  - Jest config: [`apps/frontend/jest.config.cjs`](../../../../apps/frontend/jest.config.cjs)
