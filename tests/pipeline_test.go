package tests

import (
	"context" // provides the background context passed into the pipeline
	"testing" // provides the testing.T type and test helpers

	"github.com/sx-tane/context-os/domain/contracts"                    // SourceRequest used to drive the pipeline
	"github.com/sx-tane/context-os/domain/pipelines"                    // Run function under test
	"github.com/sx-tane/context-os/internal/ingestion"                  // NewPipeline wraps connectors for the run
	githubsource "github.com/sx-tane/context-os/internal/source/github" // GitHub connector used as the test source
)

// TestRunDetectsPotentialMismatch verifies that a document containing mismatch keywords
// produces at least one extracted entity and at least one mismatch finding.
func TestRunDetectsPotentialMismatch(t *testing.T) {
	pipe := ingestion.NewPipeline(githubsource.NewConnector()) // build a pipeline with a single GitHub connector
	result, err := pipelines.Run(context.Background(), pipe, contracts.SourceRequest{
		URI:     "repo://example",                                                                   // identify the artifact being ingested
		Content: "frontend expects refundStatus but backend has missingRefundState mismatch",        // content with known mismatch keywords
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err) // fail immediately if the pipeline itself errors
	}
	if len(result.Graph.Entities) == 0 {
		t.Fatal("expected extracted entities in graph") // the content should have produced at least one entity
	}
	if len(result.Mismatches) == 0 {
		t.Fatal("expected mismatch detection") // the mismatch keyword in the content should trigger a finding
	}
}
