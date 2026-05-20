package tests

import (
	"context"
	"testing"

	"github.com/sx-tane/context-os/domain/contracts"
	"github.com/sx-tane/context-os/domain/pipelines"
	"github.com/sx-tane/context-os/internal/ingestion"
	githubsource "github.com/sx-tane/context-os/internal/source/github"
)

func TestRunDetectsPotentialMismatch(t *testing.T) {
	pipe := ingestion.NewPipeline(githubsource.NewConnector())
	result, err := pipelines.Run(context.Background(), pipe, contracts.SourceRequest{
		URI:     "repo://example",
		Content: "frontend expects refundStatus but backend has missingRefundState mismatch",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Graph.Entities) == 0 {
		t.Fatal("expected extracted entities in graph")
	}
	if len(result.Mismatches) == 0 {
		t.Fatal("expected mismatch detection")
	}
}
