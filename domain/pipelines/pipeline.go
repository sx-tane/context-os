package pipelines

import (
	"context" // carries cancellation and deadline signals through the pipeline

	"github.com/sx-tane/context-os/domain/contracts"        // SourceRequest input type
	"github.com/sx-tane/context-os/domain/types"            // Mismatch output type
	"github.com/sx-tane/context-os/internal/classification" // classifies documents by signal type
	"github.com/sx-tane/context-os/internal/extraction"     // extracts named entities from documents
	"github.com/sx-tane/context-os/internal/graph"          // builds and stores the context graph
	"github.com/sx-tane/context-os/internal/identity"       // merges aliases into canonical entities
	"github.com/sx-tane/context-os/internal/ingestion"      // fans out ingestion across connectors
	"github.com/sx-tane/context-os/internal/normalization"  // converts events to normalized documents
	"github.com/sx-tane/context-os/internal/reasoning"      // detects mismatches in the graph
	"github.com/sx-tane/context-os/internal/relationship"   // builds edges between canonical entities
)

// Result is the output produced by a full pipeline run.
type Result struct {
	Graph      *graph.ContextGraph `json:"graph"`      // all entities and relationships accumulated during the run
	Mismatches []types.Mismatch    `json:"mismatches"` // delivery misalignments detected by the reasoning stage
}

// Run executes the full pipeline: ingest → normalize → classify → extract → resolve → relate → reason.
func Run(ctx context.Context, sourcePipeline ingestion.Pipeline, req contracts.SourceRequest) (Result, error) {
	events, err := sourcePipeline.Ingest(ctx, req) // collect raw events from all registered connectors
	if err != nil {
		return Result{}, err // stop early and surface the error if any connector fails
	}
	contextGraph := graph.New() // create a fresh in-memory graph to accumulate this run's findings
	for _, event := range events { // process each ingested event through every pipeline stage in order
		doc := normalization.Normalize(event)             // convert the raw event into a canonical document
		classified := classification.Classify(doc)        // determine the document's signal type and confidence
		extracted := extraction.Extract(classified)        // pull named entities out of the document body
		canonical := identity.Resolve(extracted)           // merge duplicate names into single canonical entities
		relationships := relationship.Build(canonical)     // link entities that co-occur in the same document
		contextGraph.AddEntities(canonical)                // store the resolved entities in the graph
		contextGraph.AddRelationships(relationships)       // store the discovered relationships in the graph
	}
	return Result{Graph: contextGraph, Mismatches: reasoning.DetectMismatches(contextGraph)}, nil // return the graph and any detected mismatches
}
