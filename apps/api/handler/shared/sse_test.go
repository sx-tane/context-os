package shared

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestSSEWriterSerialisesConcurrentWrites verifies log, status, and result
// writes can be issued from multiple goroutines without a data race or
// interleaved output.  Run with -race to exercise the guarantee.
func TestSSEWriterSerialisesConcurrentWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := NewSSEWriter(rec, rec)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = sw.Write([]byte("codex log line\n"))
		}()
		go func() {
			defer wg.Done()
			sw.Event("status", `{"status":"running"}`)
		}()
	}
	wg.Wait()

	body := rec.Body.String()
	if !strings.Contains(body, "event: log") || !strings.Contains(body, "event: status") {
		t.Fatalf("expected both log and status events, got %q", body)
	}
}

// TestSSEWriterErrorAndResult verifies the JSON-framed error and result helpers
// emit the expected SSE event names and payloads.
func TestSSEWriterErrorAndResult(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := NewSSEWriter(rec, rec)

	sw.Error("invalid_request", "uri is required")
	sw.Result(map[string]string{"connector": "codex-cli"})

	body := rec.Body.String()
	for _, want := range []string{
		"event: error",
		`"error":"invalid_request"`,
		`"message":"uri is required"`,
		"event: result",
		`"connector":"codex-cli"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %q, want substring %q", body, want)
		}
	}
}
