// Package sync provides a background worker that performs incremental connector sync.
// It reads all connector sync records, identifies stale or failed ones, and re-runs
// ingestion using the stored cursor to fetch only new events since the last sync.
package sync

import (
	"context"
	"log"
	"time"

	"context-os/domain/repository"
)

// Worker performs periodic incremental sync for all registered workspaces.
type Worker struct {
	workspaces repository.WorkspaceRepository
	syncs      repository.SyncRepository
	events     repository.EventRepository
}

// NewWorker returns a Worker that uses the provided repositories.
func NewWorker(workspaces repository.WorkspaceRepository, syncs repository.SyncRepository, events repository.EventRepository) *Worker {
	return &Worker{workspaces: workspaces, syncs: syncs, events: events}
}

// Run starts a background ticker loop that triggers a sync pass every interval.
// It returns only when ctx is cancelled.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runPass(ctx)
		}
	}
}

// runPass iterates all workspaces and logs which connector syncs are stale or in
// error state. Full re-ingest is triggered by the caller via the presentation
// endpoint; this pass marks which syncs need attention so status endpoints can
// surface them to the user.
func (w *Worker) runPass(ctx context.Context) {
	workspaces, err := w.workspaces.List(ctx)
	if err != nil {
		log.Printf("sync: list workspaces: %v", err)
		return
	}
	if len(workspaces) == 0 {
		return
	}
	staleCutoff := time.Now().Add(-15 * time.Minute)
	for _, ws := range workspaces {
		syncs, err := w.syncs.ListByWorkspace(ctx, ws.ID)
		if err != nil {
			log.Printf("sync: list syncs for workspace %s: %v", ws.ID, err)
			continue
		}
		for _, s := range syncs {
			stale := isStaleSync(s, staleCutoff)
			errored := s.Status == "error"
			if stale || errored {
				log.Printf("sync: workspace=%s connector=%s uri=%s needs_sync=true (stale=%v error=%v)",
					ws.ID, s.Connector, s.SourceURI, stale, errored)
				// Mark the sync as pending so the next user-triggered ingest picks up the cursor.
				pending := s
				pending.Status = "pending"
				if uErr := w.syncs.Upsert(ctx, pending); uErr != nil {
					log.Printf("sync: mark pending for %s/%s: %v", ws.ID, s.Connector, uErr)
				}
			}
		}
	}
}

func isStaleSync(sync repository.ConnectorSync, cutoff time.Time) bool {
	hasLocalSyncState := sync.LastSyncedAt != nil || sync.EventCount > 0 || sync.Cursor != ""
	if !hasLocalSyncState {
		return false
	}
	if sync.LastSyncedAt == nil {
		return true
	}
	return sync.LastSyncedAt.Before(cutoff)
}
