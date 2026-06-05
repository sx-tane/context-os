package codexio_test

import (
	"testing"

	"context-os/internal/codexio"
)

// TestBoundedBufferRetainsOnlyRecentBytes verifies writes cannot grow retained log data past the configured limit.
func TestBoundedBufferRetainsOnlyRecentBytes(t *testing.T) {
	buffer := codexio.NewBoundedBuffer(5)

	if _, err := buffer.Write([]byte("abc")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := buffer.Write([]byte("def")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := buffer.String(); got != "bcdef" {
		t.Fatalf("String() = %q, want %q", got, "bcdef")
	}
	buffer.Reset()
	if got := buffer.String(); got != "" {
		t.Fatalf("String() after Reset() = %q, want empty", got)
	}
}

// TestBoundedBufferTrimsLargeSingleWrite verifies a large write keeps its latest bytes.
func TestBoundedBufferTrimsLargeSingleWrite(t *testing.T) {
	buffer := codexio.NewBoundedBuffer(4)

	if _, err := buffer.Write([]byte("abcdef")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := buffer.String(); got != "cdef" {
		t.Fatalf("String() = %q, want %q", got, "cdef")
	}
}
