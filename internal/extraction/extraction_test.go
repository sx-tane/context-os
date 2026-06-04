package extraction_test

import (
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/domain/types"
	"context-os/internal/extraction"
)

// TestExtractDeduplicatesTokensAndPreservesEvidenceMetadata verifies extraction returns stable candidates with classification and source provenance.
func TestExtractDeduplicatesTokensAndPreservesEvidenceMetadata(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "doc-1",
			Body: "refundStatus refundStatus missingRefundState DBColumn PaymentFlag",
			Metadata: map[string]string{
				contracts.MetadataSourceURI: "repo://refund-flow",
				events.MetadataSourceID:     "github:issue:1",
			},
		},
		Classification: types.ConsumerConcern,
	}

	got := extraction.Extract(doc)
	if len(got) != 4 {
		t.Fatalf("Extract() length = %d, want 4", len(got))
	}
	if got[0].Name != "refundStatus" || got[0].Type != types.APIField {
		t.Fatalf("first entity = %#v, want refundStatus api_field", got[0])
	}
	if got[1].Name != "missingRefundState" || got[1].Type != types.APIField {
		t.Fatalf("second entity = %#v, want missingRefundState api_field", got[1])
	}
	if got[2].Type != types.DBColumn {
		t.Fatalf("third entity type = %q, want %q", got[2].Type, types.DBColumn)
	}
	if got[3].Type != types.Enum {
		t.Fatalf("fourth entity type = %q, want %q", got[3].Type, types.Enum)
	}
	if got[0].Metadata["classification"] != string(types.ConsumerConcern) {
		t.Fatalf("Metadata[classification] = %q, want %q", got[0].Metadata["classification"], types.ConsumerConcern)
	}
	if got[0].Metadata[contracts.MetadataSourceURI] != "repo://refund-flow" {
		t.Fatalf("Metadata[source_uri] = %q, want repo://refund-flow", got[0].Metadata[contracts.MetadataSourceURI])
	}
	if got[0].Metadata[events.MetadataSourceID] != "github:issue:1" {
		t.Fatalf("Metadata[source_id] = %q, want github:issue:1", got[0].Metadata[events.MetadataSourceID])
	}
}

// TestExtractTokenPathRecordsMethodConfidenceAndSpans verifies regex extraction attaches provenance fields.
func TestExtractTokenPathRecordsMethodConfidenceAndSpans(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{ID: "doc-1", Body: "refundStatus changed"},
	}

	got := extraction.Extract(doc)
	if len(got) == 0 {
		t.Fatalf("Extract() length = 0, want at least one entity")
	}
	first := got[0]
	if first.ExtractionMethod != extraction.MethodRegexToken {
		t.Errorf("ExtractionMethod = %q, want %q", first.ExtractionMethod, extraction.MethodRegexToken)
	}
	if first.RawMention != "refundStatus" {
		t.Errorf("RawMention = %q, want refundStatus", first.RawMention)
	}
	if first.Confidence <= 0 {
		t.Errorf("Confidence = %v, want > 0", first.Confidence)
	}
	if len(first.Spans) != 1 || first.Spans[0].Field != "body" || first.Spans[0].Start != 0 {
		t.Errorf("Spans = %+v, want a single body span starting at 0", first.Spans)
	}
}

// TestExtractSkipsPlainLanguageStatusToken verifies plain-language lowercase status tokens are not fallback entities.
func TestExtractSkipsPlainLanguageStatusToken(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{ID: "doc-1", Body: "refundStatus status"},
	}

	got := extraction.Extract(doc)
	if len(got) != 1 {
		t.Fatalf("Extract() length = %d, want 1", len(got))
	}
	if got[0].Name != "refundStatus" || got[0].Type != types.APIField {
		t.Fatalf("first entity = %#v, want refundStatus api_field", got[0])
	}
}

// TestExtractRegexFallbackKeepsIdentifiersAndSkipsStopwords verifies token fallback favors code and ticket identifiers over prose.
func TestExtractRegexFallbackKeepsIdentifiersAndSkipsStopwords(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "doc-1",
			Body: "Source Read fields and also among type travel_fee_commission_target_flag travelFeeCommissionTargetFlag BA015 BKGDEV-8466",
		},
	}

	got := extraction.Extract(doc)
	names := map[string]bool{}
	for _, entity := range got {
		names[entity.Name] = true
	}
	for _, want := range []string{"travel_fee_commission_target_flag", "travelFeeCommissionTargetFlag", "BA015", "BKGDEV-8466"} {
		if !names[want] {
			t.Fatalf("Extract() missing %q in %+v", want, got)
		}
	}
	for _, noisy := range []string{"Source", "Read", "fields", "and", "also", "among", "type"} {
		if names[noisy] {
			t.Fatalf("Extract() included noisy token %q in %+v", noisy, got)
		}
	}
}

// TestExtractCodexLabelsProducesAssistiveTypedEntities verifies structured Codex labels become evidence-backed metadata.
func TestExtractCodexLabelsProducesAssistiveTypedEntities(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID: "codex-1",
			Body: `Human summary.
CONTEXTOS_LABELS_JSON: {"entities":{"requirement":[{"name":"Travel fee commission target","evidence":"Jira BKGDEV-8466 names the requirement","confidence":0.82}],"api_field":[{"name":"travelFeeCommissionTargetFlag","evidence":"API field listed in source","confidence":0.91}],"service":[{"name":"booking-service","evidence":"service owner comment","confidence":0.7}],"dependency":[],"enum":[{"name":"BA015","evidence":"status value in sheet","confidence":0.8}],"db_column":[{"name":"travel_fee_commission_target_flag","evidence":"DB column in mapping","confidence":0.88}]},"risks":[],"decisions":[],"status":[]}`,
			Metadata: map[string]string{
				contracts.MetadataSourceURI: "https://example.atlassian.net/browse/BKGDEV-8466",
			},
		},
	}

	got := extraction.Extract(doc)
	if len(got) != 5 {
		t.Fatalf("Extract() length = %d, want 5", len(got))
	}
	byName := map[string]types.Entity{}
	for _, entity := range got {
		byName[entity.Name] = entity
		if entity.ExtractionMethod != extraction.MethodCodexLabel {
			t.Errorf("ExtractionMethod = %q, want %q", entity.ExtractionMethod, extraction.MethodCodexLabel)
		}
		if entity.Metadata["assistive"] != "true" {
			t.Errorf("Metadata[assistive] = %q, want true", entity.Metadata["assistive"])
		}
		if entity.Metadata["evidence"] == "" {
			t.Errorf("Metadata[evidence] = empty for %q", entity.Name)
		}
	}
	if byName["travelFeeCommissionTargetFlag"].Type != types.APIField {
		t.Fatalf("api field type = %q, want api_field", byName["travelFeeCommissionTargetFlag"].Type)
	}
	if byName["travel_fee_commission_target_flag"].Type != types.DBColumn {
		t.Fatalf("db column type = %q, want db_column", byName["travel_fee_commission_target_flag"].Type)
	}
}

// TestExtractOpenAPIUsesPointerMetadata verifies OpenAPI pointer metadata yields typed structured entities.
func TestExtractOpenAPIUsesPointerMetadata(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "spec-1",
			Body: "{}",
			Metadata: map[string]string{
				"filesystem_format":         "openapi_spec",
				"openapi_schema_pointers":   `["/components/schemas/RefundStatus"]`,
				"openapi_enum_pointers":     `["/components/schemas/RefundStatus/enum"]`,
				"openapi_endpoint_pointers": `["/paths/~1refunds/get"]`,
			},
		},
	}

	got := extraction.Extract(doc)
	if len(got) != 3 {
		t.Fatalf("Extract() length = %d, want 3", len(got))
	}
	byMethod := map[string]types.Entity{}
	for _, e := range got {
		if e.ExtractionMethod != extraction.MethodOpenAPI {
			t.Errorf("entity %q method = %q, want %q", e.Name, e.ExtractionMethod, extraction.MethodOpenAPI)
		}
		byMethod[string(e.Type)] = e
	}
	if byMethod["service"].Name != "get" {
		t.Errorf("endpoint entity name = %q, want get", byMethod["service"].Name)
	}
	if byMethod["service"].Spans[0].Path == "" {
		t.Errorf("endpoint entity span path = empty, want JSON pointer")
	}
}

// TestExtractSpreadsheetParsesCellLines verifies spreadsheet cell content produces entities with cell-reference spans.
func TestExtractSpreadsheetParsesCellLines(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "wb-1",
			Body: "Sheet1!A1=RefundStatus\nSheet1!B2.formula===SUM(A1:A2)\nSheet1!B3=Approved",
			Metadata: map[string]string{
				"filesystem_format": "spreadsheet",
			},
		},
	}

	got := extraction.Extract(doc)
	if len(got) != 2 {
		t.Fatalf("Extract() length = %d, want 2", len(got))
	}
	if got[0].Name != "RefundStatus" || got[0].ExtractionMethod != extraction.MethodSpreadsheet {
		t.Errorf("first entity = %#v, want RefundStatus via spreadsheet", got[0])
	}
	if got[0].Spans[0].Path != "Sheet1!A1" {
		t.Errorf("first entity span path = %q, want Sheet1!A1", got[0].Spans[0].Path)
	}
}

// TestExtractJiraParsesFieldsObject verifies a Jira payload yields entities for each scalar field.
func TestExtractJiraParsesFieldsObject(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "jira-1",
			Body: `{"key":"CTX-42","fields":{"summary":"Refund state","status":"open","watches":{"count":1}}}`,
			Metadata: map[string]string{
				"connector": "jira",
			},
		},
	}

	got := extraction.Extract(doc)
	if len(got) != 2 {
		t.Fatalf("Extract() length = %d, want 2 scalar fields", len(got))
	}
	for _, e := range got {
		if e.ExtractionMethod != extraction.MethodJiraField {
			t.Errorf("entity %q method = %q, want %q", e.Name, e.ExtractionMethod, extraction.MethodJiraField)
		}
	}
}

// TestExtractGitHubFallsBackToTokensForPlainText verifies non-JSON GitHub content falls through to token extraction.
func TestExtractGitHubFallsBackToTokensForPlainText(t *testing.T) {
	doc := types.ClassifiedDocument{
		Document: types.NormalizedDocument{
			ID:   "gh-1",
			Body: "frontend expects refundStatus but backend exposes missingRefundState",
			Metadata: map[string]string{
				"connector": "github",
			},
		},
	}

	got := extraction.Extract(doc)
	if len(got) == 0 {
		t.Fatalf("Extract() length = 0, want token fallback entities")
	}
	if got[0].ExtractionMethod != extraction.MethodRegexToken {
		t.Errorf("ExtractionMethod = %q, want %q (token fallback)", got[0].ExtractionMethod, extraction.MethodRegexToken)
	}
}
