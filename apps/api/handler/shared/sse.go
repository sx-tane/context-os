package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/events"
	codexsource "context-os/internal/source/codex"
)

// SSEWriter wraps an http.ResponseWriter+http.Flusher and writes SSE-formatted
// log lines as Codex stdout/stderr bytes arrive.  Each newline-terminated chunk
// is emitted as a "log" event so the browser can append it to the live log panel.
type SSEWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

// NewSSEWriter creates an SSEWriter backed by w and f.
func NewSSEWriter(w http.ResponseWriter, f http.Flusher) *SSEWriter {
	return &SSEWriter{w: w, f: f}
}

// Write satisfies io.Writer.  Each call emits a "log" SSE event per non-empty
// line and flushes immediately so the browser receives it without buffering.
func (s *SSEWriter) Write(p []byte) (int, error) {
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

// SSEError writes an SSE error event with the given code and message.
func SSEError(w http.ResponseWriter, f http.Flusher, code, msg string) {
	b, _ := json.Marshal(map[string]string{"error": code, "message": msg})
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", b)
	f.Flush()
}

// SSEResult writes an SSE result event with v serialised as JSON.
func SSEResult(w http.ResponseWriter, f http.Flusher, v any) {
	b, _ := json.Marshal(v)
	_, _ = fmt.Fprintf(w, "event: result\ndata: %s\n\n", b)
	f.Flush()
}

// ingestResult carries the outcome of a background IngestStream call.
type ingestResult struct {
	events []events.Event
	err    error
}

// StreamWithHeartbeat runs fn in a goroutine and sends SSE "status" events
// every 2 seconds so the client knows the connection is alive and how long
// Codex has been running.  It returns when fn completes or ctx is cancelled.
func StreamWithHeartbeat(ctx context.Context, w http.ResponseWriter, f http.Flusher, fn func() ([]events.Event, error)) ([]events.Event, error) {
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
			_, _ = fmt.Fprintf(w, "event: status\ndata: %s\n\n", b)
			f.Flush()
		}
	}
}

// CodexStreamRequest constrains the request types accepted by StreamCodexIngest.
type CodexStreamRequest interface {
	request.GithubIngest | request.JiraIngest | request.SlackIngest
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
		SSEError(w, f, "invalid_json", err.Error())
		return
	}
	requestURI := strings.TrimSpace(uri(req))
	if requestURI == "" {
		SSEError(w, f, "invalid_request", "uri is required")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(provider(req)), "codex") {
		SSEError(w, f, "invalid_request", "streaming is only supported for provider=codex")
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

	resultEvents, err := StreamWithHeartbeat(ctx, w, f, func() ([]events.Event, error) {
		return codexsource.IngestStream(ctx, NewSourceRequest(requestURI, metadata), sw)
	})
	if err != nil {
		SSEError(w, f, "ingest_error", err.Error())
		return
	}
	if len(resultEvents) == 0 {
		SSEError(w, f, "empty_result", "connector returned no events")
		return
	}

	SSEResult(w, f, NewIngestResponse("codex-cli", capabilities, resultEvents))
}
