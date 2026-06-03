package presentation

import (
	"context"
	"testing"
	"time"
)

func TestPresentationWriteContextSurvivesCanceledParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	parentCancel()

	ctx, cancel := presentationWriteContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("presentationWriteContext() error = %v, want active context", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("presentationWriteContext() has no deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("presentationWriteContext() deadline = %s, want future deadline", deadline)
	}
}
