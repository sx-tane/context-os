// Package pipeline orchestrates a full pipeline run by calling each internal stage in order.
// It is the only place that is permitted to import multiple internal stage packages together.
package pipeline

import (
	"context"

	"context-os/domain/contracts"
	"context-os/domain/pipelines"
	"context-os/internal/classification"
	"context-os/internal/extraction"
	"context-os/internal/graph"
	"context-os/internal/identity"
	"context-os/internal/ingestion"
	"context-os/internal/normalization"
	"context-os/internal/reasoning"
	"context-os/internal/relationship"
)

// Run executes the full pipeline: ingest → normalize → classify → extract → resolve → relate → reason.
func Run(ctx context.Context, sourcePipeline ingestion.Pipeline, req contracts.SourceRequest) (pipelines.Result, error) {
	events, err := sourcePipeline.Ingest(ctx, req)
	if err != nil {
		return pipelines.Result{}, err
	}
	contextGraph := graph.New()
	for _, event := range events {
		doc := normalization.Normalize(event)
		classified := classification.Classify(doc)
		extracted := extraction.Extract(classified)
		canonical := identity.Resolve(extracted)
		rels := relationship.Build(canonical)
		contextGraph.AddEntities(canonical)
		contextGraph.AddRelationships(rels)
	}
	return pipelines.Result{
		Entities:   contextGraph.AllEntities(),
		Mismatches: reasoning.DetectMismatches(contextGraph),
	}, nil
}
