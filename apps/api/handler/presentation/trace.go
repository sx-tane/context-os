package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// buildTraceID creates a stable content-addressed trace ID for a findings response.
func buildTraceID(connector, uri string, mismatchIDs []string) string {
	raw := strings.ToLower(strings.TrimSpace(connector)) + "|" + strings.TrimSpace(uri) + "|" + strings.Join(mismatchIDs, ",")
	sum := sha256.Sum256([]byte(raw))
	return "trace-" + hex.EncodeToString(sum[:8])
}

// buildRunTraceID generates a unique trace ID per execution using connector,
// URI, and a nanosecond timestamp so each pipeline run has a distinct identity
// even when the same connector+URI is used repeatedly.
func buildRunTraceID(connector, uri string) string {
	raw := fmt.Sprintf("%s|%s|%d", strings.ToLower(strings.TrimSpace(connector)), strings.TrimSpace(uri), time.Now().UnixNano())
	sum := sha256.Sum256([]byte(raw))
	return "run-" + hex.EncodeToString(sum[:8])
}
