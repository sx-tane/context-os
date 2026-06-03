# API Handler Skeleton

Copy and adapt these files for a new `<name>` connector.
Replace every `<Name>` / `<name>` / `<CAPABILITY>` / `<EnvVar>` placeholder.

---

## 1. `apps/api/request/ingest.go` — add this struct

```go
// <Name>Ingest is the JSON body accepted by POST /<name>/ingest.
type <Name>Ingest struct {
	URI      string `json:"uri"      example:"<scheme>://<resource>"`
	Token    string `json:"token"    example:"<token-example>"`
	Provider string `json:"provider" example:"token"`
}
```

---

## 2. `apps/api/handler/<name>/<name>.go`

```go
// Package <name> provides HTTP handlers for the /<name>/* routes.
package <name>

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	<name>source "context-os/internal/source/<name>"
)

// Status handles GET /<name>/status.
// It reports whether the required environment variables are configured.
//
// @Summary      <Name> connection status
// @Description  Returns whether <Name> environment variables are configured.
// @Tags         <name>
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /<name>/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	token := strings.TrimSpace(os.Getenv("<EnvVar>"))
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected": token != "",
		"source":    "env",
	})
}

// Ingest handles POST /<name>/ingest by ingesting a <Name> artifact
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a <Name> artifact
// @Description  Fetches a <Name> artifact by URI and returns a provenance-rich event.
// @Tags         <name>
// @Accept       json
// @Produce      json
// @Param        body  body      request.<Name>Ingest  true  "<Name> ingest request"
// @Success      200   {object}  response.Ingest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /<name>/ingest [post]
func Ingest(w http.ResponseWriter, r *http.Request) {
	shared.RunSourceIngest(w, r, <name>source.NewConnector(),
		func(d *json.Decoder) (shared.SourceIngestInput, error) {
			var req request.<Name>Ingest
			if err := d.Decode(&req); err != nil {
				return shared.SourceIngestInput{}, err
			}
			return shared.SourceIngestInput{
				URI:  req.URI,
				Metadata: map[string]string{
					"token": req.Token,
				},
			}, nil
		},
	)
}

// IngestStream handles POST /<name>/ingest/stream.
// It streams Codex CLI output as SSE log events and emits a final "result" event.
//
// @Summary      Stream ingest a <Name> artifact via Codex
// @Description  Runs Codex plugin for <Name> and streams log lines as SSE events.
// @Tags         <name>
// @Produce      text/event-stream
// @Param        body  body  request.<Name>Ingest  true  "<Name> ingest request"
// @Success      200
// @Failure      400  {object}  map[string]string
// @Failure      405  {object}  map[string]string
// @Router       /<name>/ingest/stream [post]
func IngestStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.<Name>Ingest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	f, ok := shared.SSEHeaders(w)
	if !ok {
		return
	}
	sw := shared.NewSSEWriter(w, f)
	shared.RunSourceIngestStream(w, r, f, sw, <name>source.NewConnector(), shared.SourceIngestInput{
		URI: req.URI,
		Metadata: map[string]string{"token": req.Token},
	})
}
```

---

## 3. `apps/api/handler/<name>/<name>_test.go`

```go
package <name>_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-os/apps/api/handler/<name>"
)

// TestStatusMethodNotAllowed verifies that a non-GET request to Status returns 405.
func TestStatusMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/<name>/status", nil)

	<name>.Status(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Status() status = %d, want 405", recorder.Code)
	}
}

// TestStatusReturnsDisconnectedWhenNoEnvVar verifies that Status reports connected=false when <EnvVar> is unset.
func TestStatusReturnsDisconnectedWhenNoEnvVar(t *testing.T) {
	t.Setenv("<EnvVar>", "")

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/<name>/status", nil)

	<name>.Status(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status() status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"connected":false`) {
		t.Fatalf("body = %s, want connected:false", recorder.Body.String())
	}
}

// TestIngestMethodNotAllowed verifies that a non-POST request to Ingest returns 405.
func TestIngestMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/<name>/ingest", nil)

	<name>.Ingest(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Ingest() status = %d, want 405", recorder.Code)
	}
}

// TestIngestRejectsMalformedJSON verifies that a request with invalid JSON returns 400.
func TestIngestRejectsMalformedJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/<name>/ingest", strings.NewReader("{bad json}"))
	req.Header.Set("Content-Type", "application/json")

	<name>.Ingest(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Ingest() status = %d, want 400", recorder.Code)
	}
}

// TestIngestStreamMethodNotAllowed verifies that a non-POST request to IngestStream returns 405.
func TestIngestStreamMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/<name>/ingest/stream", nil)

	<name>.IngestStream(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("IngestStream() status = %d, want 405", recorder.Code)
	}
}
```

---

## 4. `internal/source/<name>/<name>.go`

```go
// Package <name> provides an MCP source connector for <Name> artifacts.
package <name>

import (
	"context"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

type connector struct {
	base source.MCPConnector
}

// NewConnector returns a <Name> source connector that ingests <describe artifact type> events.
func NewConnector() contracts.MCPSourceConnector {
	return connector{
		base: source.NewMCPConnector("<name>", contracts.Capability<CAPABILITY>),
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest delegates to the base MCP connector after enriching request metadata.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	req.Metadata = cloneMetadata(req.Metadata)
	// Add connector-specific metadata enrichment here.
	return c.base.Ingest(ctx, req)
}

func cloneMetadata(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
```

---

## 5. `apps/api/main.go` — add to `registerRoutes` call

```go
import (
    // existing imports...
    <name> "context-os/apps/api/handler/<name>"
)

// inside registerRoutes slice:
{pattern: "/<name>/status",        handler: http.HandlerFunc(<name>.Status),       cors: true},
{pattern: "/<name>/ingest",        handler: http.HandlerFunc(<name>.Ingest),        cors: true},
{pattern: "/<name>/ingest/stream", handler: http.HandlerFunc(<name>.IngestStream),  cors: true},
```
