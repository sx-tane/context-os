---
name: go-best-practices
description: "Apply Go code quality rules when writing, reviewing, or refactoring Go code. Use when: implementing a new Go function, struct, or interface; reviewing code for readability; refactoring existing Go files; deciding how to handle errors, goroutines, concurrency, or package organization; asking about Go naming, documentation, or API design. Covers the twelve canonical Go best practices: error handling, repetition avoidance, code ordering, documentation, naming, package structure, interfaces, concurrency, and goroutine leak prevention."
argument-hint: "Which file or pattern are you working on?"
user-invocable: true
---

# Go Best Practices

## Outcome

Go code remains simple, idiomatic, documented, and safe at package and pipeline boundaries.

## Procedure

1. Read the package boundary and public API before editing.
2. Apply the rules below while implementing or reviewing the Go change.
3. Update the nearest README when exported behavior, package structure, commands, or setup changes.
4. Run the relevant Go build, test, and vet checks for touched packages.

## Purpose

Keep every Go file in this codebase simple, readable, and maintainable by following the twelve canonical Go best practices. Apply these rules when writing new code, reviewing existing code, or refactoring.

---

## 1. Avoid Nesting — Handle Errors First

Return early on error instead of nesting inside `if err == nil` blocks.
Less nesting = less cognitive load on the reader.

```go
// BAD — nested happy path
func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    err = binary.Write(w, binary.LittleEndian, int32(len(g.Name)))
    if err == nil {
        size += 4
        // ... more nesting
    }
    return
}

// GOOD — guard clause, return on error
func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    if err := binary.Write(w, binary.LittleEndian, int32(len(g.Name))); err != nil {
        return 0, err
    }
    size += 4
    // ... flat happy path continues
    return size, nil
}
```

**Rule**: Every error path ends with `return` at the top of the block. Happy path stays at the left margin.

---

## 2. Avoid Repetition — Use Utility Types

When the same error-check pattern repeats across multiple calls, introduce a small carrier type that holds the first error and skips subsequent calls silently.

```go
type errWriter struct {
    w   io.Writer
    err error
}

func (ew *errWriter) write(v interface{}) {
    if ew.err != nil {
        return // already failed, skip
    }
    ew.err = binary.Write(ew.w, binary.LittleEndian, v)
}

// Caller becomes flat and readable
func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    ew := &errWriter{w: w}
    ew.write(int32(len(g.Name)))
    ew.write([]byte(g.Name))
    ew.write(int64(g.AgeYears))
    // single error check at the end
    return ew.n, ew.err
}
```

**Rule**: One-off utility types are fine when they eliminate repeated boilerplate. Create them local to the package; do not over-abstract.

---

## 3. Important Code Goes First

Order matters within a file. Readers scan top-to-bottom.

```
1. License header / build tags
2. Package doc comment
3. import block (stdlib, then external, blank line between groups)
4. Most significant exported types and constructors
5. Methods on those types
6. Helper types and functions last
```

```go
import (
    "fmt"
    "io"

    "golang.org/x/net/websocket"
)
```

**Rule**: Put the thing the reader came to find near the top. Helpers and internals go at the bottom.

---

## 4. Document Your Code

Every exported identifier must have a doc comment that starts with the identifier name.

```go
// Package classification assigns routing labels to normalized pipeline documents.
package classification

// Classify assigns a classification label and confidence score to the given document.
// It uses deterministic keyword matching so the result is reproducible from the same input.
func Classify(doc types.NormalizedDocument) types.ClassifiedDocument {
```

**Rule**: Package doc before `package`, exported type/func doc directly above the declaration, starting with the identifier name.

---

## 5. Shorter Names Are Better

The package name already provides context. Do not repeat it in the identifier.

| Package          | Exported name              | How it is used                               |
| ---------------- | -------------------------- | -------------------------------------------- |
| `classification` | `Classifier`               | `classification.Classifier` ✅               |
| `classification` | `ClassificationClassifier` | `classification.ClassificationClassifier` ❌ |
| `identity`       | `Resolver`                 | `identity.Resolver` ✅                       |

**Rule**: Prefer the shortest name that is self-explanatory at the call site, accounting for the package prefix.

---

## 6. Split Large Packages Into Multiple Files

- Keep individual files focused on a single concern (e.g. `parse.go`, `render.go`).
- Put tests in a companion `_test.go` file (same package or `_test` suffix for black-box tests).
- When a package has more than one file, add a `doc.go` file that holds only the package documentation comment.

```
internal/reasoning/
    doc.go           ← package doc only
    reasoning.go     ← core DetectMismatches logic
    reasoning_test.go
```

---

## 7. Ask for What You Need — Use Narrow Interfaces

Accept the smallest interface that satisfies the function's requirements. Do not accept a concrete type when an interface works.

```go
// BAD — locks caller into os.File
func (g *Gopher) WriteToFile(f *os.File) (int64, error)

// BAD — requires both Read and Write when only Write is needed
func (g *Gopher) WriteToReadWriter(rw io.ReadWriter) (int64, error)

// GOOD — accepts anything that can be written to
func (g *Gopher) WriteTo(w io.Writer) (int64, error)
```

**Rule in ContextOS**: Accept `context.Context` and narrow domain interfaces (e.g. `MCPSourceConnector`) rather than concrete structs wherever the stage boundary permits it.

---

## 8. Keep Independent Packages Independent

If two packages do not need to know about each other, keep them that way. Use a thin interface or a shared type from `domain/types` as the bridge.

```go
// BAD — drawer imports parser, creating a coupling
import "internal/parser"
func Draw(f parser.ParsedFunc) image.Image

// GOOD — drawer defines a local interface, no import needed
type Function interface { Eval(float64) float64 }
func Draw(f Function) image.Image
```

**Rule in ContextOS**: Internal stage packages (`internal/classification`, `internal/extraction`, …) must depend only on `domain/` contracts and types. They must not import each other directly.

---

## 9. Avoid Concurrency in Your Public API

Expose synchronous functions. Let the caller choose whether to run them concurrently. This makes code easier to test and reason about sequentially.

```go
// BAD — forces a goroutine on the caller
func ingestAsync(req SourceRequest, errc chan error) { go func() { errc <- ingest(req) }() }

// GOOD — synchronous; caller can go func() { errc <- Ingest(ctx, req) }(req) if needed
func Ingest(ctx context.Context, req SourceRequest) ([]Event, error)
```

---

## 10. Use Goroutines to Manage State — Communicate via Channels

When a goroutine owns mutable state, expose control through channels rather than shared memory.

```go
type Server struct{ quit chan struct{} }

func (s *Server) Stop() {
    s.quit <- struct{}{}
    <-s.quit // wait for acknowledgement
}
```

**Rule**: Share memory by communicating, not communicate by sharing memory.

---

## 11. Avoid Goroutine Leaks

A goroutine that blocks forever on a channel write is a leak. Two safe patterns:

**Pattern A — buffered channel sized to the number of goroutines:**

```go
errc := make(chan error, len(addrs)) // goroutines can always send
```

**Pattern B — quit channel so blocked goroutines can exit:**

```go
quit := make(chan struct{})
defer close(quit) // signals all goroutines to stop

go func(addr string) {
    select {
    case errc <- sendMsg(msg, addr):
    case <-quit: // unblock if caller returned early
    }
}(addr)
```

**Rule in ContextOS**: Any goroutine spawned inside a pipeline stage must have a clear exit path via context cancellation (`ctx.Done()`) or a quit channel.

---

## Quick Checklist

Before committing any Go file, verify:

- [ ] No `if err == nil` nesting — errors are handled first with early returns
- [ ] No repeated boilerplate — extracted to a utility type or helper if it appears 3+ times
- [ ] Imports grouped (stdlib / external), most important code at the top of the file
- [ ] All exported identifiers have a doc comment starting with the identifier name
- [ ] Names are as short as possible without losing meaning at the call site
- [ ] Functions accept narrow interfaces, not concrete types
- [ ] No direct imports between `internal/` stage packages
- [ ] Public API functions are synchronous — no goroutine in the signature
- [ ] Every goroutine has a clear termination path
- [ ] Nearest README updated when exported behavior, commands, package structure, or setup changed

## References

- [Twelve Go Best Practices — Francesc Campoy Flores](https://go.dev/talks/2013/bestpractices.slide)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Quick Checklist](./references/go-checklist.md) — review before marking Go implementation work complete.

## Assets

- [`assets/twelve-go-best-practices.md`](assets/twelve-go-best-practices.md) — full transcript of the original slide deck (all 12 practices with code examples)
- `assets/twelve-go-best-practices.pdf` — original PDF slide deck (drop the file here to keep it alongside the skill)
