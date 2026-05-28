package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/events"
	codexsource "context-os/internal/source/codex"
)

// sseWriter wraps an http.ResponseWriter+http.Flusher and writes SSE-formatted
// log lines as Codex stdout/stderr bytes arrive.  Each newline-terminated chunk
// is emitted as a "log" event so the browser can append it to the live log panel.
type sseWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

// Write satisfies io.Writer.  Each call emits a "log" SSE event and flushes
// immediately so the browser receives it without waiting for the response to end.
func (s *sseWriter) Write(p []byte) (int, error) {
	// Split on newlines so each line becomes a separate SSE event.
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

func sseHeaders(w http.ResponseWriter) (http.Flusher, bool) {
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

func sseError(w http.ResponseWriter, f http.Flusher, code, msg string) {
	b, _ := json.Marshal(map[string]string{"error": code, "message": msg})
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", b)
	f.Flush()
}

func sseResult(w http.ResponseWriter, f http.Flusher, v any) {
	b, _ := json.Marshal(v)
	_, _ = fmt.Fprintf(w, "event: result\ndata: %s\n\n", b)
	f.Flush()
}

// ingestResult carries the outcome of a background IngestStream call.
type ingestResult struct {
	events []events.Event
	err    error
}

// streamWithHeartbeat runs fn in a goroutine and sends SSE "status" events
// every 2 seconds so the client knows the connection is alive and how long
// Codex has been running.  It returns when fn completes.
func streamWithHeartbeat(ctx context.Context, w http.ResponseWriter, f http.Flusher, fn func() ([]events.Event, error)) ([]events.Event, error) {
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

type codexStreamRequest interface {
	request.GithubIngest | request.SlackIngest
}

func streamCodexIngest[T codexStreamRequest](
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

	f, ok := sseHeaders(w)
	if !ok {
		return
	}
	sw := &sseWriter{w: w, f: f}

	req, err := decode(json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)))
	if err != nil {
		sseError(w, f, "invalid_json", err.Error())
		return
	}
	requestURI := strings.TrimSpace(uri(req))
	if requestURI == "" {
		sseError(w, f, "invalid_request", "uri is required")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(provider(req)), "codex") {
		sseError(w, f, "invalid_request", "streaming is only supported for provider=codex")
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

	// Codex CLI buffers its stdout/stderr when running non-interactively.
	// streamWithHeartbeat sends live "status" SSE events every 2 s so the
	// browser shows elapsed time while waiting for the final output.
	resultEvents, err := streamWithHeartbeat(ctx, w, f, func() ([]events.Event, error) {
		return codexsource.IngestStream(ctx, newSourceRequest(requestURI, metadata), sw)
	})
	if err != nil {
		sseError(w, f, "ingest_error", err.Error())
		return
	}
	if len(resultEvents) == 0 {
		sseError(w, f, "empty_result", "connector returned no events")
		return
	}

	ev := resultEvents[0]
	sseResult(w, f, response.Ingest{
		Connector:    "codex-cli",
		Capabilities: capabilities,
		Event:        ev,
		Preview:      preview(ev.Content),
		Metadata:     ev.Metadata,
	})
}

// GithubIngestStream handles POST /github/ingest/stream.
// It streams Codex CLI log lines as SSE "log" events while the process runs,
// then emits a single "result" event with the final ingest payload.
// Only the "codex" provider is accepted; for the token provider use /github/ingest.
//
// @Summary      Stream GitHub Codex ingest
// @Description  Streams Codex CLI progress via SSE, then emits a result event.
// @Tags         github
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.GithubIngest  true  "GitHub ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /github/ingest/stream [post]
func GithubIngestStream(w http.ResponseWriter, r *http.Request) {
	streamCodexIngest(
		w,
		r,
		codexsource.PluginGitHub,
		[]string{"repository", "issues"},
		func(dec *json.Decoder) (request.GithubIngest, error) {
			var req request.GithubIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.GithubIngest) string { return req.URI },
		func(req request.GithubIngest) string { return req.Provider },
		func(req request.GithubIngest) string { return req.Token },
	)
}

// newSourceRequest constructs a contracts.SourceRequest with the given URI and metadata.
func newSourceRequest(uri string, metadata map[string]string) contracts.SourceRequest {
	return contracts.SourceRequest{URI: uri, Metadata: metadata}
}

// preview returns the first 500 runes of content as a display preview.
func preview(content string) string {
	runes := []rune(content)
	if len(runes) > 500 {
		return string(runes[:500]) + "…"
	}
	return content
}

// SlackIngestStream handles POST /slack/ingest/stream. It streams Codex CLI log lines as SSE "log" events while the process runs,
// then emits a single "result" event with the final ingest payload.
//
// @Summary      Stream Slack Codex ingest
// @Description  Streams Codex CLI progress via SSE, then emits a result event.
// @Tags         slack
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  request.SlackIngest  true  "Slack ingest request (provider must be codex)"
// @Success      200   {string}  string  "SSE stream"
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Router       /slack/ingest/stream [post]
func SlackIngestStream(w http.ResponseWriter, r *http.Request) {
	streamCodexIngest(
		w,
		r,
		codexsource.PluginSlack,
		[]string{"messages"},
		func(dec *json.Decoder) (request.SlackIngest, error) {
			var req request.SlackIngest
			err := dec.Decode(&req)
			return req, err
		},
		func(req request.SlackIngest) string { return req.URI },
		func(req request.SlackIngest) string { return req.Provider },
		func(req request.SlackIngest) string { return req.Token },
	)
}
