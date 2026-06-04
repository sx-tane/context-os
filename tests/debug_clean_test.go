package tests

import (
	"context"
	"fmt"
	"testing"

	"context-os/domain/contracts"
	"context-os/internal/pipeline"
	githubsource "context-os/internal/source/github"
	"context-os/internal/stages/ingestion"
)

func TestDebugClean(t *testing.T) {
	pipe := ingestion.NewPipeline(githubsource.NewConnector())
	result, err := pipeline.Run(context.Background(), pipe, contracts.SourceRequest{
		URI:     "repo://example/refund-clean",
		Content: "frontend displays refundStatus from backend status API",
		Metadata: map[string]string{
			"source_id": "github:issue:refund-clean",
			"team":      "payments",
			"trace_id":  "trace-refund-clean",
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range result.Entities {
		fmt.Printf("ENTITY id=%q name=%q type=%q method=%q\n", e.Entity.ID, e.Entity.Name, e.Entity.Type, e.Entity.ExtractionMethod)
	}
	for _, m := range result.Mismatches {
		fmt.Printf("MISMATCH id=%q type=%q summary=%q\n", m.ID, m.Type, m.Summary)
	}
}
