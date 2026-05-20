package pipelines

import (
	"context"

	"github.com/sx-tane/context-os/internal/classification"
	"github.com/sx-tane/context-os/internal/extraction"
	"github.com/sx-tane/context-os/internal/graph"
	"github.com/sx-tane/context-os/internal/identity"
	"github.com/sx-tane/context-os/internal/ingestion"
	"github.com/sx-tane/context-os/internal/normalization"
	"github.com/sx-tane/context-os/internal/reasoning"
	"github.com/sx-tane/context-os/internal/relationship"
	"github.com/sx-tane/context-os/domain/contracts"
	"github.com/sx-tane/context-os/domain/types"
)

type Result struct {
	Graph      *graph.ContextGraph `json:"graph"`
	Mismatches []types.Mismatch    `json:"mismatches"`
}

func RunMVP(ctx context.Context, sourcePipeline ingestion.Pipeline, req contracts.SourceRequest) (Result, error) {
	events, err := sourcePipeline.Ingest(ctx, req)
	if err != nil {
		return Result{}, err
	}
	contextGraph := graph.New()
	for _, event := range events {
		doc := normalization.Normalize(event)
		classified := classification.Classify(doc)
		extracted := extraction.Extract(classified)
		canonical := identity.Resolve(extracted)
		relationships := relationship.Build(canonical)
		contextGraph.AddEntities(canonical)
		contextGraph.AddRelationships(relationships)
	}
	return Result{Graph: contextGraph, Mismatches: reasoning.DetectMismatches(contextGraph)}, nil
}
