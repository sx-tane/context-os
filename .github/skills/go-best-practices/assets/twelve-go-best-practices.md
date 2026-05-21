# Twelve Go Best Practices
**Francesc Campoy Flores — Gopher at Google**
Source: https://go.dev/talks/2013/bestpractices.slide

---

## Best practices

From Wikipedia:

> "A best practice is a method or technique that has consistently shown results superior to those achieved with other means"

Techniques to write Go code that is

- simple,
- readable,
- maintainable.

---

## Some code

```go
type Gopher struct {
    Name     string
    AgeYears int
}

func (g *Gopher) WriteTo(w io.Writer) (size int64, err error) {
    err = binary.Write(w, binary.LittleEndian, int32(len(g.Name)))
    if err == nil {
        size += 4
        var n int
        n, err = w.Write([]byte(g.Name))
        size += int64(n)
        if err == nil {
            err = binary.Write(w, binary.LittleEndian, int64(g.AgeYears))
            if err == nil {
                size += 4
            }
            return
        }
        return
    }
    return
}
```

---

## 1. Avoid nesting by handling errors first

Less nesting means less cognitive load on the reader.

```go
func (g *Gopher) WriteTo(w io.Writer) (size int64, err error) {
    err = binary.Write(w, binary.LittleEndian, int32(len(g.Name)))
    if err != nil {
        return
    }
    size += 4
    n, err := w.Write([]byte(g.Name))
    size += int64(n)
    if err != nil {
        return
    }
    err = binary.Write(w, binary.LittleEndian, int64(g.AgeYears))
    if err == nil {
        size += 4
    }
    return
}
```

---

## 2. Avoid repetition when possible

Deploy one-off utility types for simpler code.

```go
type binWriter struct {
    w    io.Writer
    size int64
    err  error
}

// Write writes a value to the provided writer in little endian form.
func (w *binWriter) Write(v interface{}) {
    if w.err != nil {
        return
    }
    if w.err = binary.Write(w.w, binary.LittleEndian, v); w.err == nil {
        w.size += int64(binary.Size(v))
    }
}
```

Using `binWriter`:

```go
func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    bw := &binWriter{w: w}
    bw.Write(int32(len(g.Name)))
    bw.Write([]byte(g.Name))
    bw.Write(int64(g.AgeYears))
    return bw.size, bw.err
}
```

### Type switch to handle special cases

```go
func (w *binWriter) Write(v interface{}) {
    if w.err != nil {
        return
    }
    switch v.(type) {
    case string:
        s := v.(string)
        w.Write(int32(len(s)))
        w.Write([]byte(s))
    case int:
        i := v.(int)
        w.Write(int64(i))
    default:
        if w.err = binary.Write(w.w, binary.LittleEndian, v); w.err == nil {
            w.size += int64(binary.Size(v))
        }
    }
}

func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    bw := &binWriter{w: w}
    bw.Write(g.Name)
    bw.Write(g.AgeYears)
    return bw.size, bw.err
}
```

### Type switch with short variable declaration

```go
func (w *binWriter) Write(v interface{}) {
    if w.err != nil {
        return
    }
    switch x := v.(type) {
    case string:
        w.Write(int32(len(x)))
        w.Write([]byte(x))
    case int:
        w.Write(int64(x))
    default:
        if w.err = binary.Write(w.w, binary.LittleEndian, v); w.err == nil {
            w.size += int64(binary.Size(v))
        }
    }
}
```

### Writing everything or nothing

Buffer all writes, flush atomically:

```go
type binWriter struct {
    w   io.Writer
    buf bytes.Buffer
    err error
}

func (w *binWriter) Write(v interface{}) {
    if w.err != nil {
        return
    }
    switch x := v.(type) {
    case string:
        w.Write(int32(len(x)))
        w.Write([]byte(x))
    case int:
        w.Write(int64(x))
    default:
        w.err = binary.Write(&w.buf, binary.LittleEndian, v)
    }
}

// Flush writes any pending values into the writer if no error has occurred.
// If an error has occurred, earlier or with a write by Flush, the error is returned.
func (w *binWriter) Flush() (int64, error) {
    if w.err != nil {
        return 0, w.err
    }
    return w.buf.WriteTo(w.w)
}

func (g *Gopher) WriteTo(w io.Writer) (int64, error) {
    bw := &binWriter{w: w}
    bw.Write(g.Name)
    bw.Write(g.AgeYears)
    return bw.Flush()
}
```

---

## Function adapters

Before — repeated error handling:

```go
func init() {
    http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    err := doThis()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Printf("handling %q: %v", r.RequestURI, err)
        return
    }
    err = doThat()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Printf("handling %q: %v", r.RequestURI, err)
        return
    }
}
```

After — adapter wraps the error once:

```go
func init() {
    http.HandleFunc("/", errorHandler(betterHandler))
}

func errorHandler(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        err := f(w, r)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            log.Printf("handling %q: %v", r.RequestURI, err)
        }
    }
}

func betterHandler(w http.ResponseWriter, r *http.Request) error {
    if err := doThis(); err != nil {
        return fmt.Errorf("doing this: %v", err)
    }
    if err := doThat(); err != nil {
        return fmt.Errorf("doing that: %v", err)
    }
    return nil
}
```

---

## 3. Important code goes first

- License information, build tags, package documentation.
- Import statements — related groups separated by blank lines.

```go
import (
    "fmt"
    "io"
    "log"

    "golang.org/x/net/websocket"
)
```

The rest of the code starting with the most significant types, and ending with helper functions and types.

---

## 4. Document your code

Package name, with the associated documentation before:

```go
// Package playground registers an HTTP handler at "/compile" that
// proxies requests to the golang.org playground service.
package playground
```

Exported identifiers appear in godoc, they should be documented correctly:

```go
// Author represents the person who wrote and/or is presenting the document.
type Author struct {
    Elem []Elem
}

// TextElem returns the first text elements of the author details.
// This is used to display the author's name, job title, and company
// without the contact details.
func (p *Author) TextElem() (elems []Elem) {
```

---

## 5. Shorter is better

Or at least *longer is not always better*.

Try to find the **shortest name that is self explanatory**.

- Prefer `MarshalIndent` to `MarshalWithIndentation`.

Don't forget that the package name will appear before the identifier you chose.

- In package `encoding/json` we find the type `Encoder`, not `JSONEncoder`.
- It is referred to as `json.Encoder`.

---

## 6. Packages with multiple files

Should you split a package into multiple files?

- **Avoid very long files** — the `net/http` package contains 15734 lines in 47 files.
- **Separate code and tests** — `net/http/cookie.go` and `net/http/cookie_test.go` are both part of the `http` package. Test code is compiled **only** at test time.
- **Separated package documentation** — when more than one file exists in a package, create a `doc.go` containing the package documentation.

---

## 7. Make your packages "go get"-able

Some packages are potentially reusable, some others are not.

A package defining some network protocol might be reused, while one defining an executable command may not.

Structure reusable packages under `pkg/` and commands under `cmd/`.

---

## 8. Ask for what you need

Using a concrete type makes code difficult to test; use an interface instead. Ask only for the methods you need.

```go
// BAD — locks caller into os.File
func (g *Gopher) WriteToFile(f *os.File) (int64, error)

// BAD — requires both Read and Write when only Write is needed
func (g *Gopher) WriteToReadWriter(rw io.ReadWriter) (int64, error)

// GOOD — accepts anything that can be written to
func (g *Gopher) WriteToWriter(f io.Writer) (int64, error)
```

---

## 9. Keep independent packages independent

```go
import (
    "golang.org/x/talks/content/2013/bestpractices/funcdraw/drawer"
    "golang.org/x/talks/content/2013/bestpractices/funcdraw/parser"
)

f, err := parser.Parse(text)
if err != nil {
    log.Fatalf("parse %q: %v", text, err)
}
m := drawer.Draw(f, *width, *height, *xmin, *xmax)
err = png.Encode(os.Stdout, m)
if err != nil {
    log.Fatalf("encode image: %v", err)
}
```

Avoid dependency by using a local interface:

```go
// BAD — drawer imports parser
import "golang.org/x/talks/.../parser"
func DrawParsedFunc(f parser.ParsedFunc) image.Image

// GOOD — drawer defines its own interface, no cross-import
import "image"

// Function represents a drawable mathematical function.
type Function interface {
    Eval(float64) float64
}

// Draw draws an image showing a rendering of the passed Function.
func Draw(f Function) image.Image {
```

Testing with an interface instead of a concrete type:

```go
package drawer

import (
    "math"
    "testing"
)

type TestFunc func(float64) float64

func (f TestFunc) Eval(x float64) float64 { return f(x) }

var (
    ident = TestFunc(func(x float64) float64 { return x })
    sin   = TestFunc(math.Sin)
)

func TestDraw_Ident(t *testing.T) {
    m := Draw(ident)
    // Verify obtained image.
}
```

---

## 10. Avoid concurrency in your API

What if we want to use it sequentially?

```go
// BAD — forces goroutine on the caller, hard to use sequentially
func doConcurrently(job string, err chan error) {
    go func() {
        fmt.Println("doing job", job)
        time.Sleep(1 * time.Second)
        err <- errors.New("something went wrong!")
    }()
}
```

Expose synchronous APIs — calling them concurrently is easy:

```go
// GOOD — synchronous; caller wraps in go func() if needed
func do(job string) error {
    fmt.Println("doing job", job)
    time.Sleep(1 * time.Second)
    return errors.New("something went wrong!")
}

func main() {
    jobs := []string{"one", "two", "three"}
    errc := make(chan error)
    for _, job := range jobs {
        go func(job string) {
            errc <- do(job)
        }(job)
    }
    for _ = range jobs {
        if err := <-errc; err != nil {
            fmt.Println(err)
        }
    }
}
```

---

## 11. Use goroutines to manage state

Use a `chan` or a struct with a `chan` to communicate with a goroutine:

```go
type Server struct{ quit chan bool }

func NewServer() *Server {
    s := &Server{make(chan bool)}
    go s.run()
    return s
}

func (s *Server) run() {
    for {
        select {
        case <-s.quit:
            fmt.Println("finishing task")
            time.Sleep(time.Second)
            fmt.Println("task done")
            s.quit <- true
            return
        case <-time.After(time.Second):
            fmt.Println("running task")
        }
    }
}

func (s *Server) Stop() {
    fmt.Println("server stopping")
    s.quit <- true
    <-s.quit
    fmt.Println("server stopped")
}
```

---

## 12. Avoid goroutine leaks

### The problem

```go
func broadcastMsg(msg string, addrs []string) error {
    errc := make(chan error) // unbuffered — goroutines can block forever
    for _, addr := range addrs {
        go func(addr string) {
            errc <- sendMsg(msg, addr)
            fmt.Println("done")
        }(addr)
    }
    for _ = range addrs {
        if err := <-errc; err != nil {
            return err // remaining goroutines block on errc write — leak!
        }
    }
    return nil
}
```

If the function returns early on error, remaining goroutines block on the channel write. The channel is never garbage-collected.

### Fix A — buffered channel sized to goroutine count

```go
errc := make(chan error, len(addrs)) // goroutines can always send without blocking
```

### Fix B — quit channel so blocked goroutines can exit

```go
func broadcastMsg(msg string, addrs []string) error {
    errc := make(chan error)
    quit := make(chan struct{})

    defer close(quit) // signals all goroutines on return

    for _, addr := range addrs {
        go func(addr string) {
            select {
            case errc <- sendMsg(msg, addr):
                fmt.Println("done")
            case <-quit: // unblock if caller returned early
                fmt.Println("quit")
            }
        }(addr)
    }

    for _ = range addrs {
        if err := <-errc; err != nil {
            return err
        }
    }
    return nil
}
```

---

## Twelve best practices — summary

1. Avoid nesting by handling errors first
2. Avoid repetition when possible
3. Important code goes first
4. Document your code
5. Shorter is better
6. Packages with multiple files
7. Make your packages "go get"-able
8. Ask for what you need
9. Keep independent packages independent
10. Avoid concurrency in your API
11. Use goroutines to manage state
12. Avoid goroutine leaks

---

## Resources

- Go homepage: https://go.dev
- Go interactive tour: https://go.dev/tour/
- Lexical scanning with Go: https://www.youtube.com/watch?v=HxaD_trXwRE
- Concurrency is not parallelism: https://vimeo.com/49718712
- Go concurrency patterns: https://www.youtube.com/watch?v=f6kdp27TYZs
- Advanced Go concurrency patterns: https://www.youtube.com/watch?v=QDDwwePbDtw
