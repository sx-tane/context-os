---
description: "Use when writing or reviewing TypeScript test files for the frontend. Enforces Jest + SWC conventions: $lib/ mocking, fetch mocking, reactive setter patterns, and conformance checklist."
applyTo: "apps/frontend/src/**/*.test.ts"
---

# Frontend Test Instructions

## Skill

For a full step-by-step guide, skeletons, and a completion checklist, apply the **frontend-jest-swc-patterns** skill.

## File Layout

- All test files live in `src/lib/__tests__/<subject>.test.ts`.
- File name must mirror the source file exactly with `.test.ts` appended.

## Key Rules

- `jest.mock("$lib/…")` must be the **first statement** before any subject import — SWC hoisting depends on this.
- Never use a relative path in `jest.mock()` — always use the full `$lib/` alias.
- Use `makeResponse()` for any function that calls `res.text()` internally; a plain `{ ok }` cast is only acceptable for `probeService`-style functions that only check `res.ok`.
- Call `makeSetters()` / `makeOptions()` inside each `it` block — never share setter objects between tests.
- For function-updater setters `(current) => newValue`, simulate by calling `v("")` in the mock implementation.
- Use the filter pattern for clear-then-set assertions:
  ```typescript
  const errorCalls = s.setError.mock.calls.filter(([v]: [string]) => v !== "");
  expect(errorCalls).toHaveLength(0);
  ```
  Never use `not.toHaveBeenCalledWith(expect.any(String))` when a clearing call with `""` is expected.
- `DOMException` for `AbortError` is a Node 18+ global — no import or jsdom needed.

## Verify

Run `bun run test` from `apps/frontend/` after every change. All 30 baseline tests must pass before adding new ones.

## Documentation

- Update `apps/frontend/README.md` when test scripts, Jest/SWC setup, dependencies, or commands change.
- Update `apps/frontend/src/lib/README.md` when exported helper behavior or shared runner contracts change.
