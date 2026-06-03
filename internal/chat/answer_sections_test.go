package chat

import "testing"

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
