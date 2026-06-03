package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestPersistenceContextSurvivesCanceledParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	parentCancel()

	ctx, cancel := persistenceContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("persistenceContext() error = %v, want active context", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("persistenceContext() has no deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("persistenceContext() deadline = %s, want future deadline", deadline)
	}
}
