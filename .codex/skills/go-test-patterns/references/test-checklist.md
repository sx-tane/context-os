# Test Conformance Checklist

Use this checklist when reviewing a `_test.go` file or auditing a pull request for test style.
Every item must be satisfied before merging a test change.

---

## File Structure

- [ ] File is named `<subject>_test.go` in the same directory as the code under test.
- [ ] Package declaration follows the rule:
  - External package (`package <name>_test`) unless unexported symbols are required.
  - If internal (`package <name>`), a comment on the first line explains why.
- [ ] Imports are in two groups: stdlib first, then module-local. No blank lines inside a group.

---

## Doc Comments

- [ ] Every `func Test*` has a doc comment on the line immediately above it.
- [ ] Comment starts with the exact function name.
- [ ] Comment uses the form: `// TestName verifies <behaviour> <expected outcome>.`
- [ ] Comment describes behaviour and outcome — not test mechanics or setup steps.

---

## Helper Functions

- [ ] Every helper calls `t.Helper()` as its first line.
- [ ] Helper is lowercase (unexported).
- [ ] Helper has a doc comment starting with its name.

---

## Assertions

- [ ] `t.Fatalf` is used for all assertions in flat (single-case) tests.
- [ ] `t.Errorf` is used inside `t.Run` subtests so all cases run before failure is reported.
- [ ] Error messages follow the format:
  - Errors: `"SymbolName() error = %v"` or `"expected ..., got ..."`
  - Field checks: `"fieldName = %q, want %q"` with got before want.
- [ ] Nil-check or length-check guards appear **before** accessing result fields.
- [ ] No `_ = err` — all errors are checked.

---

## Test Shape

- [ ] Flat test used for a single happy-path or single error path.
- [ ] `t.Run` subtests used when multiple variants share the same assertion logic.
- [ ] Table-driven (`[]struct` + `for range`) used for three or more input/output pairs.
- [ ] No shared mutable state between top-level test functions.

---

## Platform

- [ ] `t.Skip(reason)` appears as the **first** statement for platform-conditional tests.
- [ ] Skip reason is a human-readable string explaining the constraint.

---

## Hygiene

- [ ] No `t.Log` calls for expected behaviour (only for debug traces).
- [ ] No comments that restate the code (`// check the status code` above an `if statusCode`).
- [ ] `t.TempDir()` used for all temporary file/directory fixtures.
- [ ] `context.Background()` used as context unless the test exercises cancellation.
- [ ] Nearest README updated when test commands, fixtures, or package behavior expectations changed.
