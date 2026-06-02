package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/events"
	codexsource "context-os/internal/source/codex"
)

// SSEWriter serialises all server-sent events written to a single
// http.ResponseWriter.  Codex stdout/stderr bytes arrive via Write while
// heartbeat, error, and result events may be emitted from another goroutine;
// the embedded mutex guarantees those writes never interleave on the wire.
type SSEWriter struct {
	mu sync.Mutex
	w  http.ResponseWriter
	f  http.Flusher
}

// NewSSEWriter creates an SSEWriter backed by w and f.
func NewSSEWriter(w http.ResponseWriter, f http.Flusher) *SSEWriter {
	return &SSEWriter{w: w, f: f}
}

// Write satisfies io.Writer.  Each call emits a "log" SSE event per non-empty
// line and flushes immediately so the browser receives it without buffering.
func (s *SSEWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		_, err := fmt.Fprintf(s.w, "event: log\ndata: %s\n\n", line)
		if err != nil {
			return 0, err
		}
	}
	s.f.Flush()
	return len(p), nil
}

// Event writes a single SSE event with the given name and pre-formatted data
// payload.  It is safe to call concurrently with Write.
func (s *SSEWriter) Event(name, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, data)
	s.f.Flush()
}

// Log writes a single "log" SSE event with the given message.
func (s *SSEWriter) Log(msg string) {
	s.Event("log", msg)
}

// Error writes an "error" SSE event carrying the given code and message.
func (s *SSEWriter) Error(code, msg string) {
	b, _ := json.Marshal(map[string]string{"error": code, "message": msg})
	s.Event("error", string(b))
}

// Result writes a "result" SSE event with v serialised as JSON.
func (s *SSEWriter) Result(v any) {
	b, _ := json.Marshal(v)
	s.Event("result", string(b))
}

// SSEHeaders sets SSE response headers on w and returns the http.Flusher.
// It writes a 500 error and returns false if the writer does not support flushing.
func SSEHeaders(w http.ResponseWriter) (http.Flusher, bool) {
	f, ok := w.(http.Flusher)
	if !ok {
		response.WriteError(w, http.StatusInternalServerError, "streaming_unsupported",
			"response writer does not support streaming")
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if present
	return f, true
}

// ingestResult carries the outcome of a background IngestStream call.
type ingestResult struct {
	events []events.Event
	err    error
}

// StreamWithHeartbeat runs fn in a goroutine and sends SSE "status" events
// every 2 seconds so the client knows the connection is alive and how long
// Codex has been running.  It returns when fn completes or ctx is cancelled.
// All status events are written through sw so they never interleave with the
// log events fn streams from its own goroutine.
func StreamWithHeartbeat(ctx context.Context, sw *SSEWriter, fn func() ([]events.Event, error)) ([]events.Event, error) {
	resultCh := make(chan ingestResult, 1)
	go func() {
		evs, err := fn()
		resultCh <- ingestResult{events: evs, err: err}
	}()

	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case res := <-resultCh:
			return res.events, res.err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			elapsed := int(time.Since(start).Seconds())
			b, _ := json.Marshal(map[string]any{"elapsed": elapsed, "status": "running"})
			sw.Event("status", string(b))
		}
	}
}

// CodexStreamRequest constrains the request types accepted by StreamCodexIngest.
type CodexStreamRequest interface {
	request.GithubIngest | request.JiraIngest | request.SlackIngest | request.GoogleDriveIngest | request.NotionIngest | request.SharePointIngest
}

// StreamCodexIngest handles POST for Codex CLI SSE streaming.  It decodes the
// request body, validates the provider, and streams Codex progress as SSE log
// events before emitting a final result event.
func StreamCodexIngest[T CodexStreamRequest](
	w http.ResponseWriter,
	r *http.Request,
	plugin string,
	capabilities []string,
	decode func(*json.Decoder) (T, error),
	uri func(T) string,
	provider func(T) string,
	token func(T) string,
	workspaceID func(T) string,
	connectorName func(T) string,
) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	f, ok := SSEHeaders(w)
	if !ok {
		return
	}
	sw := NewSSEWriter(w, f)

	req, err := decode(json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)))
	if err != nil {
		sw.Error("invalid_json", err.Error())
		return
	}
	requestURI := strings.TrimSpace(uri(req))
	if requestURI == "" {
		sw.Error("invalid_request", "uri is required")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(provider(req)), "codex") {
		sw.Error("invalid_request", "streaming is only supported for provider=codex")
		return
	}

	metadata := map[string]string{
		codexsource.MetadataPlugin: plugin,
	}
	if tok := strings.TrimSpace(token(req)); tok != "" {
		metadata[codexsource.MetadataTokenOverride] = tok
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	resultEvents, err := StreamWithHeartbeat(ctx, sw, func() ([]events.Event, error) {
		return codexsource.IngestStream(ctx, NewSourceRequest(requestURI, metadata), sw)
	})
	if err != nil {
		sw.Error("ingest_error", err.Error())
		return
	}
	if len(resultEvents) == 0 {
		sw.Error("empty_result", "connector returned no events")
		return
	}

	input := SourceIngestInput{
		WorkspaceID: strings.TrimSpace(workspaceID(req)),
		Connector:   strings.TrimSpace(connectorName(req)),
		URI:         requestURI,
		Metadata:    metadata,
	}
	if input.WorkspaceID != "" {
		service := GetPersistentIngestService()
		if service == nil {
			sw.Error("persistence_unavailable", "database-backed ingest is unavailable")
			return
		}
		persisted, err := service.PersistEvents(ctx, input, capabilities, resultEvents)
		if err != nil {
			sw.Error("persist_error", err.Error())
			return
		}
		sw.Result(persisted)
		return
	}

	result := NewIngestResponse("codex-cli", capabilities, resultEvents)
	result.PersistenceMode = "preview_debug"
	sw.Result(result)
}
