package presentation

import (
	"context"
	"context-os/domain/repository"
	"fmt"
	"time"
)

// logAudit writes best-effort audit events without changing user-facing findings behavior.
func (h *Handler) logAudit(ctx context.Context, event repository.AuditEvent) {
	if h == nil || h.audit == nil {
		return
	}
	if event.Actor == "" {
		event.Actor = "api"
	}
	writeCtx, cancel := presentationWriteContext(ctx)
	defer cancel()
	if err := h.audit.Log(writeCtx, event); err != nil {
		// Audit rows are operational history; a failed write should not change
		// findings semantics or hide the user-facing result.
		fmt.Printf("presentation: audit %s: %v\n", event.EventType, err)
	}
}

// presentationWriteContext creates a detached bounded context for post-response persistence writes.
func presentationWriteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), presentationWriteTimeout)
}

// lastSyncTime returns the most recent LastSyncedAt across all syncs for the given connector,
// or zero time if none found.
func lastSyncTime(syncs []repository.ConnectorSync, connector string) time.Time {
	var t time.Time
	for _, s := range syncs {
		if s.Connector != connector {
			continue
		}
		if s.LastSyncedAt != nil && s.LastSyncedAt.After(t) {
			t = *s.LastSyncedAt
		}
	}
	return t
}
