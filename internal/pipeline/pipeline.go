// Package pipeline orchestrates a full pipeline run by calling each internal stage in order.
// It is the only place that is permitted to import multiple internal stage packages together.
package pipeline

import (
	"context"
	"log"

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
}

// Run executes the full pipeline: ingest → normalize → classify → extract → resolve → relate → reason.
// If stores is non-nil and WorkspaceID is set, ingested events, entities, relationships, and mismatches are
// persisted after the in-memory run completes.
func Run(ctx context.Context, sourcePipeline ingestion.Pipeline, req contracts.SourceRequest, stores *Stores) (pipelines.Result, error) {
	rawEvents, err := sourcePipeline.Ingest(ctx, req)
	if err != nil {
		return pipelines.Result{}, err
	}
	var semanticMatcher identity.Matcher
	if stores != nil {
		semanticMatcher = stores.SemanticMatcher // nil is safe: ResolveWithMatcher falls back to LocalMatcher
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
		rels := relationship.Build(canonical)
		contextGraph.AddEntities(canonical)
		contextGraph.AddRelationships(rels)
	}
	result := pipelines.Result{
		EventCount:    len(rawEvents),
		Entities:      contextGraph.AllEntities(),
		Relationships: contextGraph.AllRelationships(),
		Mismatches:    reasoning.DetectMismatches(contextGraph),
	}

	if stores != nil && stores.WorkspaceID != "" {
		persistResult(ctx, stores, req, rawEvents, contextGraph, result)
	}
	return result, nil
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
				Connector:     e.Metadata[contracts.MetadataConnector],
				SourceURI:     req.URI,
				EventType:     string(e.Type),
				Title:         e.Content,
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
		if n, err := stores.Events.UpsertBatch(ctx, stores.WorkspaceID, ingestEvents); err != nil {
			log.Printf("pipeline: persist events: %v", err)
		} else {
			log.Printf("pipeline: persisted %d new events for workspace %s", n, stores.WorkspaceID)
		}
	}

	if stores.Entities != nil {
		if err := stores.Entities.UpsertEntities(ctx, stores.WorkspaceID, result.Entities); err != nil {
			log.Printf("pipeline: persist entities: %v", err)
		}
		if err := stores.Entities.UpsertRelationships(ctx, stores.WorkspaceID, result.Relationships); err != nil {
			log.Printf("pipeline: persist relationships: %v", err)
		}
	}

	if stores.Mismatches != nil {
		if err := stores.Mismatches.UpsertMismatches(ctx, stores.WorkspaceID, result.Mismatches, stores.TraceID); err != nil {
			log.Printf("pipeline: persist mismatches: %v", err)
		}
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
