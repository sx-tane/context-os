package chat

// White-box tests cover deterministic answer rendering helpers split out of chat.go.

import (
	"strings"
	"testing"
	"time"

	"context-os/domain/repository"
)

// TestBuildArtifactAnswerIncludesHighlights verifies local artifact answers include compact evidence highlights.
func TestBuildArtifactAnswerIncludesHighlights(t *testing.T) {
	t.Parallel()

	answer, summary := buildArtifactAnswer(Result{
		Connector: "github",
		Artifacts: []repository.IngestEvent{{
			Title:      "Fix checkout flow",
			Body:       "Release blocked because payment retry now keeps linkedFlag.\nSecond detail should also be available.",
			IngestedAt: time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC),
		}},
	}, 20, "en")
	if !strings.Contains(answer, "Found 1 local github artifacts") {
		t.Fatalf("answer = %q, want artifact count", answer)
	}
	if !strings.Contains(answer, "payment retry now keeps linkedFlag") {
		t.Fatalf("answer = %q, want compact highlight", answer)
	}
	if !strings.Contains(summary, "Fix checkout flow") {
		t.Fatalf("summary = %q, want artifact title", summary)
	}
}

// TestBuildFanoutAnswerCombinesSections verifies multi-source live sections are synthesized into one answer.
func TestBuildFanoutAnswerCombinesSections(t *testing.T) {
	t.Parallel()

	answer := buildFanoutAnswer([]AnswerSection{{
		SourceLabel: "GitHub repo",
		Connector:   "github",
		Summary:     "The checkout flow owns linkedFlag updates.",
		Facts:       []string{"linkedFlag means the payment was attached to a booking record"},
		OpenItems:   []string{"Confirm rollout date"},
	}}, []string{"jira: timeout"}, "en")
	for _, want := range []string{"Meaning:", "Open items:", "Evidence sources: GitHub 1", "failed or returned no usable answer"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer = %q, want %q", answer, want)
		}
	}
}

// TestConnectorDisplayNameFormatsKnownConnectors verifies connector labels are human readable in answers.
func TestConnectorDisplayNameFormatsKnownConnectors(t *testing.T) {
	t.Parallel()

	if got := connectorDisplayName("googledrive"); got != "Google Drive" {
		t.Fatalf("connectorDisplayName() = %q, want Google Drive", got)
	}
}
