// Package codexio provides small IO helpers for Codex CLI integration.
package codexio

// DefaultLogLimit is the maximum retained Codex stdout/stderr bytes per buffer.
const DefaultLogLimit = 1 << 20

// BoundedBuffer retains only the most recent bytes written to it.
type BoundedBuffer struct {
	limit int
	data  []byte
}

// NewBoundedBuffer returns a buffer capped at limit bytes.
func NewBoundedBuffer(limit int) *BoundedBuffer {
	if limit <= 0 {
		limit = DefaultLogLimit
	}
	return &BoundedBuffer{limit: limit}
}

// Write appends p and discards older bytes when the limit is exceeded.
func (b *BoundedBuffer) Write(p []byte) (int, error) {
	if len(p) >= b.limit {
		b.data = append(b.data[:0], p[len(p)-b.limit:]...)
		return len(p), nil
	}
	overflow := len(b.data) + len(p) - b.limit
	if overflow > 0 {
		copy(b.data, b.data[overflow:])
		b.data = b.data[:len(b.data)-overflow]
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

// String returns the retained buffer content as text.
func (b *BoundedBuffer) String() string {
	return string(b.data)
}

// Reset clears retained buffer content.
func (b *BoundedBuffer) Reset() {
	b.data = b.data[:0]
}
