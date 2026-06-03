package shared

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/domain/pipelines"
	"context-os/domain/repository"
	"context-os/internal/identity"
	"context-os/internal/ingestion"
	"context-os/internal/normalization"
	"context-os/internal/pipeline"
)

// PersistentIngestTimeout allows Codex-backed source reads to complete while
// still bounding user-triggered ingest calls.
const PersistentIngestTimeout = 5 * time.Minute
const persistentWriteTimeout = 30 * time.Second

const metadataProductConnector = "product_connector"

var persistentIngest struct {
	mu      sync.RWMutex
	service *PersistentIngestService
}

// PersistentIngestService owns the production ingest path. When a request
// includes workspace_id, handlers route through this service so source events,
// graph state, connector sync rows, and audit rows agree.
type PersistentIngestService struct {
	workspaces      repository.WorkspaceRepository
	events          repository.EventRepository
	entities        repository.EntityRepository
	mismatches      repository.MismatchRepository
	syncs           repository.SyncRepository
	audit           repository.AuditRepository
	parsedWriter    *normalization.DocumentWriter
	semanticMatcher identity.Matcher
}

// PersistentIngestOption configures optional persistence helpers.
type PersistentIngestOption func(*PersistentIngestService)

// WithPersistentParsedWriter persists normalized documents to storage/parsed.
func WithPersistentParsedWriter(w *normalization.DocumentWriter) PersistentIngestOption {
	return func(s *PersistentIngestService) { s.parsedWriter = w }
}

// WithPersistentSemanticMatcher enables semantic identity matching during ingest.
func WithPersistentSemanticMatcher(m identity.Matcher) PersistentIngestOption {
	return func(s *PersistentIngestService) { s.semanticMatcher = m }
}

// NewPersistentIngestService returns a DB-backed production ingest service.
func NewPersistentIngestService(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	entities repository.EntityRepository,
	mismatches repository.MismatchRepository,
	syncs repository.SyncRepository,
	audit repository.AuditRepository,
	opts ...PersistentIngestOption,
) *PersistentIngestService {
	s := &PersistentIngestService{
		workspaces: workspaces,
		events:     events,
		entities:   entities,
		mismatches: mismatches,
		syncs:      syncs,
		audit:      audit,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetPersistentIngestService installs the production ingest service used by
// shared handlers. Passing nil disables DB-backed ingest.
func SetPersistentIngestService(service *PersistentIngestService) {
	persistentIngest.mu.Lock()
	defer persistentIngest.mu.Unlock()
	persistentIngest.service = service
}

// GetPersistentIngestService returns the currently installed persistent service.
func GetPersistentIngestService() *PersistentIngestService {
	persistentIngest.mu.RLock()
	defer persistentIngest.mu.RUnlock()
	return persistentIngest.service
}

// Ingest reads a source connector and persists the full pipeline result.
func (s *PersistentIngestService) Ingest(ctx context.Context, connector contracts.MCPSourceConnector, input SourceIngestInput) (response.Ingest, error) {
	req := contracts.SourceRequest{
		URI:      strings.TrimSpace(input.URI),
		Content:  input.Content,
		Cursor:   strings.TrimSpace(input.Cursor),
		Metadata: withProductConnector(input.Metadata, normalizedConnectorName(input.Connector, connector.Name())),
	}
	connectorName := normalizedConnectorName(input.Connector, connector.Name())
	workspace, traceID, err := s.prepare(ctx, input.WorkspaceID, connectorName, req.URI, req.Cursor)
	if err != nil {
		return response.Ingest{}, err
	}

	stores := s.pipelineStores(workspace.ID, traceID)
	result, err := pipeline.Run(ctx, ingestion.NewPipeline(connector), req, stores)
	if err != nil {
		s.recordFailure(ctx, workspace.ID, connectorName, req.URI, traceID, err)
		return response.Ingest{}, err
	}
	if len(result.Events) == 0 {
		err := fmt.Errorf("connector returned no events")
		s.recordFailure(ctx, workspace.ID, connectorName, req.URI, traceID, err)
		return response.Ingest{}, err
	}

	return s.complete(ctx, workspace.ID, connectorName, req, traceID, CapabilityStrings(connector.Capabilities()), result), nil
}

// PersistEvents persists events that have already been emitted by a streaming connector.
func (s *PersistentIngestService) PersistEvents(
	ctx context.Context,
	input SourceIngestInput,
	capabilities []string,
	rawEvents []events.Event,
) (response.Ingest, error) {
	processCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), PersistentIngestTimeout)
	defer cancel()

	connectorName := normalizedConnectorName(input.Connector, "")
	req := contracts.SourceRequest{
		URI:      strings.TrimSpace(input.URI),
		Content:  input.Content,
		Cursor:   strings.TrimSpace(input.Cursor),
		Metadata: withProductConnector(input.Metadata, connectorName),
	}
	workspace, traceID, err := s.prepare(processCtx, input.WorkspaceID, connectorName, req.URI, req.Cursor)
	if err != nil {
		return response.Ingest{}, err
	}

	result := pipeline.RunEvents(processCtx, rawEvents, req, s.pipelineStores(workspace.ID, traceID))
	if len(result.Events) == 0 {
		err := fmt.Errorf("connector returned no events")
		s.recordFailure(processCtx, workspace.ID, connectorName, req.URI, traceID, err)
		return response.Ingest{}, err
	}
	return s.complete(processCtx, workspace.ID, connectorName, req, traceID, capabilities, result), nil
}

// PersistEvidenceEvents stores already-answered live chat evidence as local
// artifacts without running graph, identity, or findings derivation.
func (s *PersistentIngestService) PersistEvidenceEvents(
	ctx context.Context,
	input SourceIngestInput,
	capabilities []string,
	rawEvents []events.Event,
) (response.Ingest, error) {
	processCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), PersistentIngestTimeout)
	defer cancel()

	connectorName := normalizedConnectorName(input.Connector, "")
	req := contracts.SourceRequest{
		URI:      strings.TrimSpace(input.URI),
		Content:  input.Content,
		Cursor:   strings.TrimSpace(input.Cursor),
		Metadata: withProductConnector(input.Metadata, connectorName),
	}
	workspace, traceID, err := s.prepare(processCtx, input.WorkspaceID, connectorName, req.URI, req.Cursor)
	if err != nil {
		return response.Ingest{}, err
	}
	if len(rawEvents) == 0 {
		err := fmt.Errorf("connector returned no events")
		s.recordFailure(processCtx, workspace.ID, connectorName, req.URI, traceID, err)
		return response.Ingest{}, err
	}
	if s.events != nil {
		writeCtx, cancel := s.writeContext(processCtx)
		_, err := s.events.UpsertBatch(writeCtx, workspace.ID, ingestEventsFromRaw(workspace.ID, connectorName, req.URI, rawEvents))
		cancel()
		if err != nil {
			s.recordFailure(processCtx, workspace.ID, connectorName, req.URI, traceID, err)
			return response.Ingest{}, err
		}
	}
	return s.completeEvidence(processCtx, workspace.ID, connectorName, req, traceID, capabilities, rawEvents), nil
}

func withProductConnector(metadata map[string]string, connectorName string) map[string]string {
	out := make(map[string]string, len(metadata)+1)
	for key, value := range metadata {
		out[key] = value
	}
	if name := strings.TrimSpace(connectorName); name != "" {
		out[metadataProductConnector] = strings.ToLower(name)
	}
	return out
}

func (s *PersistentIngestService) prepare(ctx context.Context, workspacePath, connectorName, sourceURI, cursor string) (repository.Workspace, string, error) {
	if s == nil || s.workspaces == nil {
		return repository.Workspace{}, "", fmt.Errorf("persistent ingest is not configured")
	}
	path := strings.TrimSpace(workspacePath)
	if path == "" {
		return repository.Workspace{}, "", fmt.Errorf("workspace_id is required")
	}
	writeCtx, cancel := s.writeContext(ctx)
	ws, err := s.workspaces.Upsert(writeCtx, repository.Workspace{Name: path, Path: path})
	cancel()
	if err != nil {
		return repository.Workspace{}, "", fmt.Errorf("persistent ingest: upsert workspace: %w", err)
	}
	traceID := buildIngestTraceID(connectorName, sourceURI)
	s.logAudit(ctx, repository.AuditEvent{
		WorkspaceID: ws.ID,
		EventType:   "ingest.started",
		Actor:       "api",
		Connector:   connectorName,
		SourceURI:   sourceURI,
		TraceID:     traceID,
		Payload:     map[string]string{"cursor": cursor},
	})
	if s.syncs != nil {
		writeCtx, cancel := s.writeContext(ctx)
		_ = s.syncs.Upsert(writeCtx, repository.ConnectorSync{
			WorkspaceID: ws.ID,
			Connector:   connectorName,
			SourceURI:   sourceURI,
			Cursor:      cursor,
			Status:      "syncing",
		})
		cancel()
	}
	return ws, traceID, nil
}

func (s *PersistentIngestService) pipelineStores(workspaceID, traceID string) *pipeline.Stores {
	return &pipeline.Stores{
		WorkspaceID:     workspaceID,
		TraceID:         traceID,
		Events:          s.events,
		Entities:        s.entities,
		Mismatches:      s.mismatches,
		ParsedWriter:    s.parsedWriter,
		SemanticMatcher: s.semanticMatcher,
	}
}

func (s *PersistentIngestService) complete(ctx context.Context, workspaceID, connectorName string, req contracts.SourceRequest, traceID string, capabilities []string, result pipelines.Result) response.Ingest {
	persistedEventCount := result.EventCount
	if s.events != nil {
		writeCtx, cancel := s.writeContext(ctx)
		if count, err := s.events.Count(writeCtx, workspaceID, connectorName); err == nil {
			persistedEventCount = count
		}
		cancel()
	}

	now := time.Now().UTC()
	if s.syncs != nil {
		writeCtx, cancel := s.writeContext(ctx)
		_ = s.syncs.Upsert(writeCtx, repository.ConnectorSync{
			WorkspaceID:  workspaceID,
			Connector:    connectorName,
			SourceURI:    strings.TrimSpace(req.URI),
			Cursor:       strings.TrimSpace(req.Cursor),
			LastSyncedAt: &now,
			EventCount:   persistedEventCount,
			Status:       "idle",
		})
		cancel()
	}

	payload := map[string]string{
		"event_count":           fmt.Sprintf("%d", result.EventCount),
		"persisted_event_count": fmt.Sprintf("%d", persistedEventCount),
		"entity_count":          fmt.Sprintf("%d", len(result.Entities)),
		"relationship_count":    fmt.Sprintf("%d", len(result.Relationships)),
		"mismatch_count":        fmt.Sprintf("%d", len(result.Mismatches)),
	}
	s.logAudit(ctx, repository.AuditEvent{
		WorkspaceID: workspaceID,
		EventType:   "ingest.completed",
		Actor:       "api",
		Connector:   connectorName,
		SourceURI:   strings.TrimSpace(req.URI),
		TraceID:     traceID,
		Payload:     payload,
	})
	s.logAudit(ctx, repository.AuditEvent{
		WorkspaceID: workspaceID,
		EventType:   "graph.updated",
		Actor:       "api",
		Connector:   connectorName,
		SourceURI:   strings.TrimSpace(req.URI),
		TraceID:     traceID,
		Payload: map[string]string{
			"entity_count":       fmt.Sprintf("%d", len(result.Entities)),
			"relationship_count": fmt.Sprintf("%d", len(result.Relationships)),
			"relationship_density": relationshipDensity(
				len(result.Entities),
				len(result.Relationships),
			),
		},
	})
	if len(result.Mismatches) > 0 {
		s.logAudit(ctx, repository.AuditEvent{
			WorkspaceID: workspaceID,
			EventType:   "findings.detected",
			Actor:       "api",
			Connector:   connectorName,
			SourceURI:   strings.TrimSpace(req.URI),
			TraceID:     traceID,
			Payload: map[string]string{
				"mismatch_count": fmt.Sprintf("%d", len(result.Mismatches)),
			},
		})
	}

	ingest := NewIngestResponse(connectorName, capabilities, result.Events)
	ingest.PersistenceMode = "database"
	ingest.WorkspaceID = workspaceID
	ingest.PersistedEventCount = persistedEventCount
	ingest.EntityCount = len(result.Entities)
	ingest.RelationshipCount = len(result.Relationships)
	ingest.MismatchCount = len(result.Mismatches)
	return ingest
}

func (s *PersistentIngestService) completeEvidence(ctx context.Context, workspaceID, connectorName string, req contracts.SourceRequest, traceID string, capabilities []string, rawEvents []events.Event) response.Ingest {
	persistedEventCount := len(rawEvents)
	if s.events != nil {
		writeCtx, cancel := s.writeContext(ctx)
		if count, err := s.events.Count(writeCtx, workspaceID, connectorName); err == nil {
			persistedEventCount = count
		}
		cancel()
	}

	now := time.Now().UTC()
	if s.syncs != nil {
		writeCtx, cancel := s.writeContext(ctx)
		_ = s.syncs.Upsert(writeCtx, repository.ConnectorSync{
			WorkspaceID:  workspaceID,
			Connector:    connectorName,
			SourceURI:    strings.TrimSpace(req.URI),
			Cursor:       strings.TrimSpace(req.Cursor),
			LastSyncedAt: &now,
			EventCount:   persistedEventCount,
			Status:       "idle",
		})
		cancel()
	}

	s.logAudit(ctx, repository.AuditEvent{
		WorkspaceID: workspaceID,
		EventType:   "ingest.completed",
		Actor:       "api",
		Connector:   connectorName,
		SourceURI:   strings.TrimSpace(req.URI),
		TraceID:     traceID,
		Payload: map[string]string{
			"event_count":           fmt.Sprintf("%d", len(rawEvents)),
			"persisted_event_count": fmt.Sprintf("%d", persistedEventCount),
			"evidence_kind":         "live_chat_answer",
		},
	})

	ingest := NewIngestResponse(connectorName, capabilities, rawEvents)
	ingest.PersistenceMode = "database"
	ingest.WorkspaceID = workspaceID
	ingest.PersistedEventCount = persistedEventCount
	return ingest
}

func ingestEventsFromRaw(workspaceID, connectorName, sourceURI string, rawEvents []events.Event) []repository.IngestEvent {
	ingestEvents := make([]repository.IngestEvent, 0, len(rawEvents))
	for _, e := range rawEvents {
		ie := repository.IngestEvent{
			ID:            e.Metadata[events.MetadataEventID],
			WorkspaceID:   workspaceID,
			Connector:     connectorName,
			SourceURI:     sourceURI,
			EventType:     string(e.Type),
			Title:         eventTitleFromRaw(e),
			Body:          e.Content,
			ContentHash:   e.Metadata["content_hash"],
			Metadata:      e.Metadata,
			SchemaVersion: e.SchemaVersion,
		}
		if ie.ID == "" {
			ie.ID = e.ID
		}
		ingestEvents = append(ingestEvents, ie)
	}
	return ingestEvents
}

func eventTitleFromRaw(e events.Event) string {
	if subject := strings.TrimSpace(e.Subject); subject != "" {
		return subject
	}
	return Preview(e.Content)
}

func (s *PersistentIngestService) recordFailure(ctx context.Context, workspaceID, connectorName, sourceURI, traceID string, err error) {
	if s.syncs != nil {
		writeCtx, cancel := s.writeContext(ctx)
		_ = s.syncs.Upsert(writeCtx, repository.ConnectorSync{
			WorkspaceID: workspaceID,
			Connector:   connectorName,
			SourceURI:   sourceURI,
			Status:      "error",
			LastError:   err.Error(),
		})
		cancel()
	}
	s.logAudit(ctx, repository.AuditEvent{
		WorkspaceID: workspaceID,
		EventType:   "ingest.failed",
		Actor:       "api",
		Connector:   connectorName,
		SourceURI:   sourceURI,
		TraceID:     traceID,
		Payload:     map[string]string{"error": err.Error()},
	})
}

func (s *PersistentIngestService) logAudit(ctx context.Context, event repository.AuditEvent) {
	if s == nil || s.audit == nil {
		return
	}
	if event.Actor == "" {
		event.Actor = "api"
	}
	writeCtx, cancel := s.writeContext(ctx)
	defer cancel()
	if err := s.audit.Log(writeCtx, event); err != nil {
		log.Printf("persistent ingest: audit %s: %v", event.EventType, err)
	}
}

func (s *PersistentIngestService) writeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), persistentWriteTimeout)
}

func normalizedConnectorName(preferred, fallback string) string {
	name := strings.ToLower(strings.TrimSpace(preferred))
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(fallback))
	}
	return name
}

func buildIngestTraceID(connector, uri string) string {
	raw := fmt.Sprintf("%s|%s|%d", strings.ToLower(strings.TrimSpace(connector)), strings.TrimSpace(uri), time.Now().UnixNano())
	sum := sha256.Sum256([]byte(raw))
	return "ingest-" + hex.EncodeToString(sum[:8])
}

func relationshipDensity(entityCount, relationshipCount int) string {
	if entityCount < 2 {
		return "0"
	}
	possible := entityCount * (entityCount - 1) / 2
	return fmt.Sprintf("%.4f", float64(relationshipCount)/float64(possible))
}
