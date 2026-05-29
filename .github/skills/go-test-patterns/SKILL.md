---
name: go-test-patterns
description: "Apply the canonical ContextOS Go test pattern when writing or reviewing test files. Use when: adding a new _test.go file; reviewing an existing test for style conformance; adding test functions to existing files; choosing between flat vs table-driven vs subtest structure. Covers package declaration, doc comment format, assertion style, helper design, and table-driven conventions."
argument-hint: "Which file or stage are you testing?"
user-invocable: true
---

# ContextOS Go Test Patterns

## Outcome

Go tests are consistent, readable, isolated, and easy to review across packages.

## Procedure

1. Choose the right test shape: flat, subtest, or table-driven.
2. Start new files from [Test Skeleton](./assets/test-skeleton.md).
3. Apply the package, comment, assertion, helper, and hygiene rules below.
4. Run the [Conformance Checklist](./references/test-checklist.md) and update the nearest README when test commands, fixtures, or behavior expectations change.

## Purpose

Every `_test.go` file in this repo must follow one canonical pattern so tests are readable,
searchable, and easy to extend without style debates.

---

## 1. Package Declaration

| Situation                     | Package name                                       |
| ----------------------------- | -------------------------------------------------- |
| Testing only exported symbols | `package <name>_test` (black-box, preferred)       |
| Test needs unexported symbols | `package <name>` (white-box, justify in a comment) |
| Integration tests in `tests/` | `package tests`                                    |
| `apps/api/main_test.go`       | `package main` (needs `registerRoutes`)            |

**Rule:** Default to the external `_test` package. Switch to internal only when the test cannot
be written otherwise, and add a line comment explaining why.

---

## 2. Doc Comment on Every Test Function

Every `func Test*` must have a doc comment immediately above it with the form:

```
// TestFunctionName verifies <one-sentence description of the tested behaviour and expected outcome>.
```

- Start with the exact function name.
- One sentence only. No period needed but keep it precise.
- Describe the **behaviour under test** and the **expected result**, not the test mechanics.

**Good:**

```go
// TestIngestDerivesRepositoryMetadataFromURI verifies repository URIs produce stable GitHub repository metadata.
func TestIngestDerivesRepositoryMetadataFromURI(t *testing.T) {
```

**Bad (describes mechanics, not behaviour):**

```go
// TestIngestDerivesRepositoryMetadataFromURI calls Ingest and checks the metadata map.
func TestIngestDerivesRepositoryMetadataFromURI(t *testing.T) {
```

---

## 3. Assertion Style

### Flat (single-case) tests — use `t.Fatalf`

```go
if err != nil {
    t.Fatalf("Ingest() error = %v", err)
}
if len(events) != 1 {
    t.Fatalf("expected 1 event, got %d", len(events))
}
if events[0].Content != "expected" {
    t.Fatalf("Content = %q, want %q", events[0].Content, "expected")
}
```

- Always guard on `err != nil` or length checks **before** accessing result fields.
- Fatal format: `"SymbolName() error = %v"` for errors; `"fieldName = got, want expected"` for field checks.

### Table-driven and subtest loops — use `t.Errorf`

```go
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        got, err := fn(tc.input)
        if err != nil {
            t.Fatalf("fn() error = %v", err)
        }
        if got != tc.want {
            t.Errorf("fn(%q) = %q, want %q", tc.input, got, tc.want)
        }
    })
}
```

- Use `t.Errorf` inside `t.Run` so all subtests execute even when one fails.
- Reserve `t.Fatalf` inside subtests for fatal pre-conditions (e.g., unexpected error).

---

## 4. Test Helper Functions

```go
// helperName <one-sentence description of what it creates or returns>.
func helperName(t *testing.T, ...) ReturnType {
    t.Helper()
    // ...
}
```

- First line must be `t.Helper()` so failure lines point to the caller.
- Lowercase name — helpers are unexported.
- Doc comment starts with the function name.

---

## 5. Platform Skips

Place platform guards as the **first statement** in the test:

```go
func TestConnectorUsesCodexExecOutput(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("shell script fake command is unix-only")
    }
    // ...
}
```

---

## 6. Imports

Three groups separated by blank lines:

```go
import (
    "context"      // stdlib
    "testing"

    "context-os/domain/contracts"   // module-local
    "context-os/internal/source"
)
```

No blank line inside a group.

---

## 7. When to Use Table-Driven vs Flat vs Subtests

| Shape                                   | Use when                                                              |
| --------------------------------------- | --------------------------------------------------------------------- |
| Flat                                    | Single happy-path or single error path.                               |
| `t.Run` subtests                        | Multiple variants of the **same behaviour** (e.g., methods, formats). |
| Table-driven (`[]struct` + `for range`) | Three or more input/output pairs with shared assertion logic.         |

---

## 8. What NOT to Do

- Do not add `t.Log` for expected behaviour — logs are for debugging only.
- Do not share mutable state between test functions.
- Do not add comments that restate the assertion (`// check status code` above an `if` for status code).
- Do not use `_ = err` to discard errors.

## README Alignment

Update the nearest README when test commands, fixture locations, harness expectations, or package behavior assumptions change.

---

## References

- [Test Skeleton](./assets/test-skeleton.md) — copy-paste starting point for a new `_test.go` file.
- [Conformance Checklist](./references/test-checklist.md) — use when reviewing a test file for style.
- Real examples in this repo:
  - Black-box connector test: [`internal/source/github/github_test.go`](../../../../internal/source/github/github_test.go)
  - White-box with helpers: [`internal/source/codex/codex_test.go`](../../../../internal/source/codex/codex_test.go)
  - Subtest / table pattern: [`apps/api/middleware/cors_test.go`](../../../../apps/api/middleware/cors_test.go)
  - Integration test: [`tests/pipeline_test.go`](../../../../tests/pipeline_test.go)
