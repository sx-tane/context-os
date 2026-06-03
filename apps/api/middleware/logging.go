package middleware

import (
	"log"
	"net/http"
	"os"
	"time"
)

// WithRequestLogging logs request start and completion details for one route when enabled.
func WithRequestLogging(pattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestLoggingEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		started := time.Now()
		requestID := r.Header.Get("X-ContextOS-Request-ID")
		if requestID == "" {
			requestID = "-"
		}
		query := r.URL.RawQuery
		if query == "" {
			query = "-"
		}
		log.Printf(
			"http request start: id=%s method=%s path=%s route=%s query=%s remote=%s",
			requestID,
			r.Method,
			r.URL.Path,
			pattern,
			query,
			r.RemoteAddr,
		)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf(
			"http request done: id=%s method=%s path=%s route=%s status=%d bytes=%d duration=%s",
			requestID,
			r.Method,
			r.URL.Path,
			pattern,
			rec.status,
			rec.bytes,
			time.Since(started).Round(time.Millisecond),
		)
	})
}

func requestLoggingEnabled() bool {
	return os.Getenv("CONTEXTOS_API_REQUEST_LOGS") == "1"
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
