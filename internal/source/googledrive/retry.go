package googledrive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// backoffDuration chooses Retry-After when present, otherwise uses a small exponential retry delay.
func backoffDuration(attempt int, headers http.Header) time.Duration {
	if headers != nil {
		if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" {
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				return seconds
			}
		}
	}
	return time.Duration(1<<(attempt-1)) * 200 * time.Millisecond
}

// readResponseBody reads a Google API response while enforcing the package response-size limit.
func readResponseBody(resp *http.Response) ([]byte, error) {
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	return body, nil
}

// sleepContext waits for a retry delay while still respecting context cancellation.
func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
