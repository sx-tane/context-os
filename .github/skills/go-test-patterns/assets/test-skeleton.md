# Test Skeleton

Copy this as the starting point for a new `_test.go` file.
Replace all `<placeholder>` values. Delete sections that do not apply.

---

## Black-box (external package) — default choice

```go
package <name>_test

import (
	"context"
	"testing"

	"context-os/domain/contracts"
	<name>source "context-os/internal/source/<name>"
)

// Test<Name>HappyPath verifies <describe expected behaviour and outcome in one sentence>.
func Test<Name>HappyPath(t *testing.T) {
	// Arrange
	connector := <name>source.NewConnector()
	req := contracts.SourceRequest{
		URI:     "<scheme>://<resource>",
		Content: "<test content>",
	}

	// Act
	events, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	// Assert
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Content != "<expected content>" {
		t.Fatalf("Content = %q, want %q", events[0].Content, "<expected content>")
	}
}

// Test<Name>RejectsInvalidInput verifies <describe rejection behaviour in one sentence>.
func Test<Name>RejectsInvalidInput(t *testing.T) {
	connector := <name>source.NewConnector()

	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: ""})
	if err == nil {
		t.Fatal("Ingest() error = nil, want error for empty URI")
	}
}
```

---

## White-box (internal package) — only when unexported symbols are needed

Add a one-line comment explaining why internal access is required.

```go
// Package <name> — internal tests access newConnector to inject a fake CLI command.
package <name>

import (
	"context"
	"testing"

	"context-os/domain/contracts"
)

// helper<Name> <one-sentence description>.
func helper<Name>(t *testing.T, arg string) *connector {
	t.Helper()
	// build and return the test fixture
	return newConnector(arg, t.TempDir())
}

// Test<Name>UsesOutput verifies <describe expected behaviour in one sentence>.
func Test<Name>UsesOutput(t *testing.T) {
	c := helper<Name>(t, "<fake-arg>")

	events, err := c.Ingest(context.Background(), contracts.SourceRequest{
		URI: "<scheme>://<resource>",
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
```

---

## Subtest / table pattern — three or more variants of the same behaviour

```go
// Test<Name>HandlesFormats verifies <describe what all variants share in one sentence>.
func Test<Name>HandlesFormats(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"markdown", "file.md", "text"},
		{"json",     "file.json", "json"},
		{"csv",      "file.csv", "spreadsheet"},
	}

	connector := <name>source.NewConnector()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: tc.input})
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}
			if got := events[0].Metadata["<name>_format"]; got != tc.want {
				t.Errorf("<name>_format = %q, want %q", got, tc.want)
			}
		})
	}
}
```

---

## HTTP handler flat test pattern

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler"
)

// Test<Handler>MethodNotAllowed verifies <Handler> rejects non-<METHOD> requests with 405.
func Test<Handler>MethodNotAllowed(t *testing.T) {
	r := httptest.NewRequest(http.Method<Wrong>, "/<route>", nil)
	w := httptest.NewRecorder()

	handler.<Handler>(w, r)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Result().StatusCode)
	}
}

// Test<Handler>InvalidJSON verifies <Handler> returns 400 for malformed request bodies.
func Test<Handler>InvalidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/<route>", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handler.<Handler>(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}
```
