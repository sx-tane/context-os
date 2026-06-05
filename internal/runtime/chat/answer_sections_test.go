package chat

import (
	"strings"
	"testing"
)

// TestParseLiveAnswerMapsStructuredSections verifies Codex JSON output becomes a plain answer plus source sections.
func TestParseLiveAnswerMapsStructuredSections(t *testing.T) {
	raw := `{
		"answer": "Two sources are relevant.",
		"answer_sections": [
			{
				"source_label": "Google Drive · mapping.xlsx",
				"connector": "googledrive",
				"source_uri": "https://docs.google.com/spreadsheets/d/abc/edit",
				"summary": "Mapping is pending confirmation.",
				"facts": ["Column A maps to field_a"],
				"open_items": ["Confirm enum/value"],
				"coding_notes": ["Do not rename field_a"],
				"links": ["https://docs.google.com/spreadsheets/d/abc/edit"],
				"timestamps": ["2026-06-03T01:02:03Z"],
				"confidence": 0.82,
				"status": "open"
			}
		]
	}`

	answer, sections := parseLiveAnswer(raw, "googledrive", "googledrive")

	if answer != "Two sources are relevant." {
		t.Fatalf("answer = %q, want structured answer", answer)
	}
	if len(sections) != 1 {
		t.Fatalf("sections length = %d, want 1", len(sections))
	}
	section := sections[0]
	if section.SourceLabel != "Google Drive · mapping.xlsx" {
		t.Fatalf("SourceLabel = %q, want source label", section.SourceLabel)
	}
	if section.Connector != "googledrive" {
		t.Fatalf("Connector = %q, want googledrive", section.Connector)
	}
	if section.SourceURI != "https://docs.google.com/spreadsheets/d/abc/edit" {
		t.Fatalf("SourceURI = %q, want Google Drive URL", section.SourceURI)
	}
	if len(section.Facts) != 1 || section.Facts[0] != "Column A maps to field_a" {
		t.Fatalf("Facts = %#v, want mapped fact", section.Facts)
	}
}

// TestParseLiveAnswerKeepsPlainText verifies non-JSON Codex output remains a backward-compatible answer.
func TestParseLiveAnswerKeepsPlainText(t *testing.T) {
	answer, sections := parseLiveAnswer("Plain source answer.", "github", "sx-tane/context-os")

	if answer != "Plain source answer." {
		t.Fatalf("answer = %q, want plain text", answer)
	}
	if len(sections) != 0 {
		t.Fatalf("sections length = %d, want 0", len(sections))
	}
}

// TestBuildFanoutAnswerSynthesizesSections verifies multi-source live answers summarize behavior instead of listing sources only.
func TestBuildFanoutAnswerSynthesizesSections(t *testing.T) {
	sections := []AnswerSection{
		{
			SourceLabel: "forcia/kkg_payment linked-flag.ts",
			Connector:   "github",
			Summary:     "Defines the two legal linkedFlag values used by transaction history.",
			Facts: []string{
				"LinkedFlag.NOT_LINKED is '0'.",
				"LinkedFlag.LINKED is '1'.",
			},
		},
		{
			SourceLabel: "forcia/kkg_payment transaction-history.repository-dynamodb.ts",
			Connector:   "github",
			Summary:     "Implements DynamoDB query, update, batch read, retry, and validation behavior for linkedFlag.",
			Facts: []string{
				"If linkedFlag is undefined, the KeyConditionExpression uses transaction_date only; otherwise it uses transaction_date AND linked_flag.",
				"updateLinkedFlag updates each matching primary key with UpdateExpression 'SET linked_flag = :linkedFlag' and value '1'.",
			},
		},
		{
			SourceLabel: "forcia/kkg_payment PR #367",
			Connector:   "github",
			Summary:     "Merged PR that made omitted linkedFlag mean full-day extraction for recovery/re-run cases.",
			Facts: []string{
				"PR #367 changed omitted linkedFlag behavior so reruns/recovery can fetch the whole day.",
			},
		},
		{
			SourceLabel: "BKGDEV-8528 Slack thread",
			Connector:   "slack",
			Summary:     "Recent PR thread for permanent mitigation of DynamoDB ProvisionedThroughputExceededException.",
			OpenItems:   []string{"PR #379 is open and unmerged, so treat it as proposed behavior."},
		},
	}

	answer := buildFanoutAnswer(sections, nil, "en")

	for _, want := range []string{"Summary:", "Meaning:", "How it works:", "Change history/current status:", "Open items:", "Evidence sources:"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer = %q, want %q section", answer, want)
		}
	}
	if strings.Contains(answer, "forcia/kkg_payment linked-flag.ts: Defines") {
		t.Fatalf("answer = %q, should not be source-list-only output", answer)
	}
}
