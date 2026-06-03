# Frontend Test Conformance Checklist

Use this checklist when reviewing a `*.test.ts` file or auditing a pull request.
Every item must be satisfied before merging a test change.

---

## File Structure

- [ ] File is named `<subject>.test.ts` (matches source file name exactly).
- [ ] File lives in `src/lib/__tests__/`.
- [ ] `jest.mock("$lib/…")` calls appear **before** the subject `import` (SWC hoisting requirement).
- [ ] Mock calls use the full `$lib/` alias path — no relative `../` in `jest.mock()`.

---

## Imports

- [ ] Three sections in order: jest.mock calls → subject import → mocked re-imports.
- [ ] Mocked module is re-imported after the subject and cast to `jest.Mock` or
      `jest.MockedFunction<typeof fn>`.

---

## global.fetch Mocking

- [ ] `fetchMock` is declared at module scope with the correct generic signature
      `jest.fn<Promise<Response>, [RequestInfo | URL, RequestInit?]>()`.
- [ ] `global.fetch` is assigned via `(global as unknown as Record<string, unknown>).fetch`.
- [ ] `fetchMock.mockReset()` is called in `beforeEach`.
- [ ] `makeResponse()` helper is used for functions that call `res.text()` internally.
      A plain `{ ok: true }` cast is acceptable only for functions that only check `res.ok`.

---

## Reactive Setter Mocking

- [ ] A `makeSetters()` / `makeOptions()` builder function is defined at module scope.
- [ ] Each test calls the builder to get a fresh set — no shared setter objects.
- [ ] Function-updater form `(current: T) => T` is handled in mock implementations
      by calling `v("")` (or the appropriate initial value) and inspecting the result.

---

## Describe / It Structure

- [ ] Outer `describe` name: `"<functionName> — <scenario group>"`.
- [ ] `it` description: present tense, describes behaviour and expected outcome in one sentence.
- [ ] Related cases grouped in one `describe` block (e.g. all token-provider cases together).

---

## Assertions

- [ ] No `expect(setter).not.toHaveBeenCalledWith(expect.any(String))` when the setter is
      called with `""` for state-clearing — use the filter pattern instead:
  ```typescript
  const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
  expect(errorCalls).toHaveLength(0);
  ```
- [ ] Lifecycle setter (setLoading / setRunning) uses call-sequence capture to assert
      `calls[0] === true` and `calls[calls.length - 1] === false`.
- [ ] `not.toHaveBeenCalled()` is used instead of `toHaveBeenCalledTimes(0)`.
- [ ] `jest.clearAllMocks()` is called in `beforeEach` for tests that use module mocks.

---

## Abort Error Handling

- [ ] AbortError test creates the error as `new DOMException("aborted", "AbortError")`.
- [ ] No `jest-environment-jsdom` required — `DOMException` is a Node 18+ global.
- [ ] Test asserts that no non-empty error setter call exists after the abort path.

---

## isCurrent Guard

- [ ] Guard test passes `isCurrent: () => false` and asserts that **no** setter was called.
- [ ] Guard test is in its own `describe` block named `"— isCurrent guard"`.

---

## Hygiene

- [ ] `jest.clearAllMocks()` or `mockReset()` in `beforeEach` — no stale mock state between tests.
- [ ] No `console.log` / `console.error` in test code.
- [ ] No direct file system or network access — everything is mocked.
- [ ] `bun run test` passes with exit code 0 after the change.
- [ ] `bun run test:coverage` shows no unexpected coverage gaps in `src/lib/` utilities.

## Documentation

- [ ] `apps/frontend/README.md` updated when test scripts, Jest/SWC setup, dependencies, or commands change.
- [ ] `apps/frontend/src/lib/README.md` updated when exported helper behavior, shared runner contracts, or utility coverage changes.
- [ ] `.github/README.md` updated when the canonical frontend test pattern or customization routing changes.
