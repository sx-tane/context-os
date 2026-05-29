package classification_test

import (
	"testing"

	"context-os/domain/types"
	"context-os/internal/classification"
)

// TestClassifyAppliesRulePriorityAndConfidence verifies keyword rules produce stable categories and confidence scores.
func TestClassifyAppliesRulePriorityAndConfidence(t *testing.T) {
	cases := []struct {
		name           string
		body           string
		classification types.Classification
		confidence     float64
	}{
		{name: "blocker wins over api", body: "api rollout is blocked", classification: types.Blocker, confidence: 0.9},
		{name: "frontend concern", body: "frontend screen expects refundStatus", classification: types.ConsumerConcern, confidence: 0.75},
		{name: "producer concern", body: "backend service layer owns refundStatus", classification: types.ProducerConcern, confidence: 0.75},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classification.Classify(types.NormalizedDocument{Body: tc.body})
			if got.Classification != tc.classification {
				t.Errorf("Classification = %q, want %q", got.Classification, tc.classification)
			}
			if got.Confidence != tc.confidence {
				t.Errorf("Confidence = %v, want %v", got.Confidence, tc.confidence)
			}
		})
	}
}

// TestClassifyDefaultsUnknown verifies unmatched documents receive the unknown category with low confidence.
func TestClassifyDefaultsUnknown(t *testing.T) {
	got := classification.Classify(types.NormalizedDocument{Body: "plain status update"})
	if got.Classification != types.Unknown {
		t.Fatalf("Classification = %q, want %q", got.Classification, types.Unknown)
	}
	if got.Confidence != 0.4 {
		t.Fatalf("Confidence = %v, want 0.4", got.Confidence)
	}
}
