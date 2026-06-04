// Package pipeline orchestrates a full pipeline run by calling each internal stage in order.
// It is the only place that is permitted to import multiple internal stage packages together.
package pipeline

import (
	"context"
	"log"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/entities"
	"context-os/domain/events"
	"context-os/domain/pipelines"
	"context-os/domain/repository"
	"context-os/internal/classification"
	"context-os/internal/extraction"
	"context-os/internal/graph"
	"context-os/internal/identity"
	"context-os/internal/ingestion"
	"context-os/internal/normalization"
	"context-os/internal/reasoning"
	"context-os/internal/relationship"
)

const metadataProductConnector = "product_connector"
const persistTimeout = 30 * time.Second

// Stores groups the repository interfaces that pipeline.Run uses to persist
// its output.  All fields are optional — a nil field disables that persistence
// path without changing run semantics.
type Stores struct {
	// WorkspaceID is the stable identifier of the workspace this run belongs to.
	// Required when any of the repository fields are non-nil.
	WorkspaceID string
	// TraceID is the pipeline trace identifier, used to correlate persisted findings.
	TraceID string
	// Events stores the raw ingested source events.
	Events repository.EventRepository
	// Entities stores canonical entities and their relationships.
	Entities repository.EntityRepository
	// Mismatches stores reasoning findings.
	Mismatches repository.MismatchRepository
	// ParsedWriter, when non-nil, persists NormalizedDocuments to storage/parsed/
	// for offline replay and debug inspection.
	ParsedWriter *normalization.DocumentWriter
	// SemanticMatcher, when non-nil, enables the Layer-2 semantic identity pass
	// using embedding cosine similarity alongside the default deterministic layers.
	SemanticMatcher identity.Matcher
	// RelationshipAssistant, when non-nil, can add validated same-document
	// relationship edges after deterministic relationship rules run.
	RelationshipAssistant relationship.Assistant
}

// Run executes the full pipeline: ingest → normalize → classify → extract → resolve → relate → reason.
// If stores is non-nil and WorkspaceID is set, ingested events, entities, relationships, and mismatches are
// persisted after the in-memory run completes.
func Run(ctx context.Context, sourcePipeline ingestion.Pipeline, req contracts.SourceRequest, stores *Stores) (pipelines.Result, error) {
	rawEvents, err := sourcePipeline.Ingest(ctx, req)
	if err != nil {
		return pipelines.Result{}, err
	}
	return RunEvents(ctx, rawEvents, req, stores), nil
}

// RunEvents executes the non-ingest pipeline stages for events that were already
// read by a connector. This keeps streaming ingest persistence on the same path
// as synchronous ingest without rerunning the source connector.
func RunEvents(ctx context.Context, rawEvents []events.Event, req contracts.SourceRequest, stores *Stores) pipelines.Result {
	return runEvents(ctx, rawEvents, req, stores, true)
}

// RunEventsGraphOnly executes normalization, classification, extraction,
// identity, relationship, and graph persistence for events already returned by a
// connector. It intentionally skips reasoning so live chat evidence can update
// Activity and Graph without auto-producing Findings.
func RunEventsGraphOnly(ctx context.Context, rawEvents []events.Event, req contracts.SourceRequest, stores *Stores) pipelines.Result {
	return runEvents(ctx, rawEvents, req, stores, false)
}

func runEvents(ctx context.Context, rawEvents []events.Event, req contracts.SourceRequest, stores *Stores, includeReasoning bool) pipelines.Result {
	var semanticMatcher identity.Matcher
	var relationshipAssistant relationship.Assistant
	if stores != nil {
		semanticMatcher = stores.SemanticMatcher // nil is safe: ResolveWithMatcher falls back to LocalMatcher
		relationshipAssistant = stores.RelationshipAssistant
	}

	contextGraph := graph.New()
	for _, event := range rawEvents {
		doc := normalization.Normalize(event)
		if stores != nil && stores.ParsedWriter != nil {
			if err := stores.ParsedWriter.Write(stores.WorkspaceID, doc); err != nil {
				log.Printf("pipeline: write parsed doc %s: %v", doc.ID, err)
			}
		}
		classified := classification.Classify(doc)
		extracted := extraction.Extract(classified)
		var canonical []entities.CanonicalEntity
		if semanticMatcher != nil {
			canonical = identity.ResolveWithMatcher(extracted, semanticMatcher, identity.MatchOptions{})
		} else {
			canonical = identity.Resolve(extracted)
		}
		rels := relationship.BuildWithAssist(ctx, doc, canonical, relationshipAssistant)
		contextGraph.AddEntities(canonical)
		contextGraph.AddRelationships(rels)
	}
	result := pipelines.Result{
		EventCount:    len(rawEvents),
		Events:        rawEvents,
		Entities:      contextGraph.AllEntities(),
		Relationships: contextGraph.AllRelationships(),
	}
	if includeReasoning {
		result.Mismatches = reasoning.DetectMismatches(contextGraph)
	}

	if stores != nil && stores.WorkspaceID != "" {
		persistResult(ctx, stores, req, rawEvents, contextGraph, result)
	}
	return result
}

// persistResult writes pipeline outputs to the backing store and saves a filesystem snapshot.
// Errors are logged and do not fail the caller — pipeline semantics are unchanged by storage failures.
func persistResult(ctx context.Context, stores *Stores, req contracts.SourceRequest, rawEvents []events.Event, contextGraph *graph.ContextGraph, result pipelines.Result) {
	if stores.Events != nil {
		ingestEvents := make([]repository.IngestEvent, 0, len(rawEvents))
		for _, e := range rawEvents {
			ie := repository.IngestEvent{
				ID:            e.Metadata[events.MetadataEventID],
				WorkspaceID:   stores.WorkspaceID,
				Connector:     persistedConnector(e.Metadata),
				SourceURI:     req.URI,
				EventType:     string(e.Type),
				Title:         eventTitle(e),
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
		persistCtx, cancel := persistenceContext(ctx)
		if n, err := stores.Events.UpsertBatch(persistCtx, stores.WorkspaceID, ingestEvents); err != nil {
			cancel()
			log.Printf("pipeline: persist events: %v", err)
		} else {
			cancel()
			log.Printf("pipeline: persisted %d event rows for workspace %s", n, stores.WorkspaceID)
		}
	}

	if stores.Entities != nil {
		persistCtx, cancel := persistenceContext(ctx)
		if err := stores.Entities.UpsertEntities(persistCtx, stores.WorkspaceID, result.Entities); err != nil {
			log.Printf("pipeline: persist entities: %v", err)
		}
		cancel()

		persistCtx, cancel = persistenceContext(ctx)
		if err := stores.Entities.UpsertRelationships(persistCtx, stores.WorkspaceID, result.Relationships); err != nil {
			log.Printf("pipeline: persist relationships: %v", err)
		}
		cancel()
	}

	if stores.Mismatches != nil {
		persistCtx, cancel := persistenceContext(ctx)
		if err := stores.Mismatches.UpsertMismatches(persistCtx, stores.WorkspaceID, result.Mismatches, stores.TraceID); err != nil {
			log.Printf("pipeline: persist mismatches: %v", err)
		}
		cancel()
	}

	// Save a deterministic filesystem snapshot so organizational memory is
	// available offline and for replay even without Postgres.
	snapshotName := stores.WorkspaceID
	if stores.TraceID != "" {
		snapshotName = stores.WorkspaceID + "_" + stores.TraceID
	}
	if _, err := contextGraph.SaveSnapshot("storage/snapshots", snapshotName); err != nil {
		log.Printf("pipeline: save snapshot: %v", err)
	}
}

func persistenceContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), persistTimeout)
}

func persistedConnector(metadata map[string]string) string {
	if connector := strings.TrimSpace(metadata[metadataProductConnector]); connector != "" {
		return strings.ToLower(connector)
	}
	return strings.ToLower(strings.TrimSpace(metadata[contracts.MetadataConnector]))
}

func eventTitle(e events.Event) string {
	if subject := strings.TrimSpace(e.Subject); subject != "" {
		return subject
	}
	return previewTitle(e.Content, 120)
}

func previewTitle(text string, limit int) string {
	preview := strings.Join(strings.Fields(text), " ")
	if len(preview) <= limit {
		return preview
	}
	if limit <= 3 {
		return preview[:limit]
	}
	return preview[:limit-3] + "..."
}
