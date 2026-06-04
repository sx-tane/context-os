package classification_test

import (
	"testing"

	"context-os/domain/types"
	"context-os/internal/stages/classification"
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
	if len(got.Labels) != 0 {
		t.Fatalf("Labels length = %d, want 0", len(got.Labels))
	}
	if len(got.MatchedRules) != 0 {
		t.Fatalf("MatchedRules length = %d, want 0", len(got.MatchedRules))
	}
}

// TestClassifyRecordsPrimaryEvidenceAndMatchedRule verifies the primary label carries matched-keyword evidence and a rule name.
func TestClassifyRecordsPrimaryEvidenceAndMatchedRule(t *testing.T) {
	got := classification.Classify(types.NormalizedDocument{Body: "this work is blocked"})

	if got.Classification != types.Blocker {
		t.Fatalf("Classification = %q, want %q", got.Classification, types.Blocker)
	}
	if len(got.Evidence) != 1 || got.Evidence[0] != "blocked" {
		t.Fatalf("Evidence = %v, want [blocked]", got.Evidence)
	}
	if len(got.MatchedRules) != 1 || got.MatchedRules[0] != "blocker_keyword" {
		t.Fatalf("MatchedRules = %v, want [blocker_keyword]", got.MatchedRules)
	}
}

// TestClassifyEmitsAllMatchingLabelsOrderedByConfidence verifies multi-signal documents retain every matching label ranked by confidence.
func TestClassifyEmitsAllMatchingLabelsOrderedByConfidence(t *testing.T) {
	got := classification.Classify(types.NormalizedDocument{Body: "api rollout is blocked"})

	if got.Classification != types.Blocker {
		t.Fatalf("Classification = %q, want %q", got.Classification, types.Blocker)
	}
	if len(got.Labels) != 2 {
		t.Fatalf("Labels length = %d, want 2", len(got.Labels))
	}
	if got.Labels[0].Classification != types.Blocker {
		t.Errorf("Labels[0] = %q, want blocker", got.Labels[0].Classification)
	}
	if got.Labels[1].Classification != types.APIDiscussion {
		t.Errorf("Labels[1] = %q, want api_discussion", got.Labels[1].Classification)
	}
	if got.Labels[0].Confidence < got.Labels[1].Confidence {
		t.Errorf("labels not ordered by confidence: %v then %v", got.Labels[0].Confidence, got.Labels[1].Confidence)
	}
}
