# Go Best Practices Checklist

Use before marking Go implementation or review work complete.

## Error Handling

- [ ] Errors are handled first with guard clauses.
- [ ] Happy-path logic stays at the left margin.
- [ ] No `if err == nil` nesting.

## Package Shape

- [ ] Imports are grouped by standard library, external, and module-local packages.
- [ ] Important exported types, constructors, and functions appear before helpers.
- [ ] Packages with multiple files have package documentation.

## API Design

- [ ] Exported identifiers have doc comments starting with the identifier name.
- [ ] Names are short and do not repeat the package name unnecessarily.
- [ ] Functions accept narrow interfaces instead of concrete types when possible.

## ContextOS Boundaries

- [ ] Internal stage packages do not import each other directly.
- [ ] Stage APIs are synchronous; callers decide when to use goroutines.
- [ ] Every goroutine has a `ctx.Done()` or quit-channel exit path.

## Documentation And Validation

- [ ] Nearest README updated when exported behavior, commands, package structure, or setup changed.
- [ ] Relevant `go test` packages pass.
- [ ] Relevant `go vet` packages pass.
