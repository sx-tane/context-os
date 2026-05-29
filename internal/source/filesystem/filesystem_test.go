package filesystem_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"context-os/domain/contracts"
	"context-os/domain/events"
	filesystemsource "context-os/internal/source/filesystem"
)

// TestNewConnectorExposesFilesystemCapability verifies the filesystem connector identity and capability.
func TestNewConnectorExposesFilesystemCapability(t *testing.T) {
	connector := filesystemsource.NewConnector()

	if connector.Name() != "filesystem" {
		t.Fatalf("expected connector name filesystem, got %q", connector.Name())
	}
	capabilities := connector.Capabilities()
	if len(capabilities) != 1 || capabilities[0] != contracts.CapabilityFiles {
		t.Fatalf("expected files capability, got %#v", capabilities)
	}
}

// TestIngestReadsTextFileWithStableHashMetadata verifies local files emit path, hash, and replay metadata.
func TestIngestReadsTextFileWithStableHashMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requirements.md")
	if err := os.WriteFile(path, []byte("# Refund Requirements\nrefundStatus must stay aligned\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if !strings.Contains(event.Content, "refundStatus") {
		t.Fatalf("expected file text content, got %q", event.Content)
	}
	if event.Metadata["path"] != path {
		t.Fatalf("path metadata = %q, want %q", event.Metadata["path"], path)
	}
	if event.Metadata["filesystem_format"] != "text" || event.Metadata["filesystem_extension"] != "md" {
		t.Fatalf("unexpected format metadata: %#v", event.Metadata)
	}
	if event.Metadata["filesystem_content_hash"] == "" || event.Metadata[contracts.MetadataSourceCursor] == "" {
		t.Fatalf("expected content hash and cursor metadata: %#v", event.Metadata)
	}

	replayed, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("replay ingest returned error: %v", err)
	}
	if replayed[0].ID != event.ID {
		t.Fatalf("expected replay-stable event ID %q, got %q", event.ID, replayed[0].ID)
	}

	if err := os.WriteFile(path, []byte("# Refund Requirements\nrefundState changed\n"), 0600); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	changed, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("changed ingest returned error: %v", err)
	}
	if changed[0].ID != event.ID {
		t.Fatalf("expected artifact ID to remain stable across content changes")
	}
	if changed[0].Metadata["filesystem_content_hash"] == event.Metadata["filesystem_content_hash"] {
		t.Fatalf("expected changed file hash to differ")
	}
}

// TestIngestRecursivelyReadsFolderWithStableChildEvents verifies folder ingestion emits deterministic file events.
func TestIngestRecursivelyReadsFolderWithStableChildEvents(t *testing.T) {
	root := t.TempDir()
	archiveDir := filepath.Join(root, "archive")
	docsDir := filepath.Join(root, "docs")
	for _, dir := range []string{archiveDir, docsDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("create directory %s: %v", dir, err)
		}
	}
	files := map[string]string{
		filepath.Join(docsDir, "b.md"):      "B requirement\n",
		filepath.Join(docsDir, "a.md"):      "A requirement\n",
		filepath.Join(archiveDir, "old.md"): "archived requirement\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("write file %s: %v", path, err)
		}
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: root,
		Metadata: map[string]string{
			"filesystem_include": "*.md",
			"filesystem_exclude": "archive",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	if len(ingested) != 2 {
		t.Fatalf("expected 2 folder file events, got %d", len(ingested))
	}

	firstPath := filepath.Join(docsDir, "a.md")
	secondPath := filepath.Join(docsDir, "b.md")
	if ingested[0].Subject != firstPath || ingested[1].Subject != secondPath {
		t.Fatalf("expected deterministic lexical event order, got %q then %q", ingested[0].Subject, ingested[1].Subject)
	}
	metadata := ingested[0].Metadata
	if metadata["filesystem_ingest_mode"] != "folder" || metadata["filesystem_root"] != root {
		t.Fatalf("expected folder metadata, got %#v", metadata)
	}
	if metadata["filesystem_relative_path"] != "docs/a.md" {
		t.Fatalf("filesystem_relative_path = %q, want docs/a.md", metadata["filesystem_relative_path"])
	}
	if metadata["filesystem_folder_file_count"] != "2" || metadata["filesystem_folder_skipped_count"] != "1" {
		t.Fatalf("unexpected folder counts: %#v", metadata)
	}
	if ingested[0].SourceID != "filesystem:file:"+firstPath {
		t.Fatalf("source_id = %q, want filesystem file source", ingested[0].SourceID)
	}
	if metadata[contracts.MetadataObjectType] != "file" || metadata[contracts.MetadataObjectID] != firstPath {
		t.Fatalf("unexpected file identity metadata: %#v", metadata)
	}

	replayed, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: root,
		Metadata: map[string]string{
			"filesystem_include": "*.md",
			"filesystem_exclude": "archive",
		},
	})
	if err != nil {
		t.Fatalf("replay ingest returned error: %v", err)
	}
	if replayed[0].ID != ingested[0].ID || replayed[1].ID != ingested[1].ID {
		t.Fatalf("expected replay-stable folder event IDs")
	}
}

// TestIngestFolderReportsPartialFailures verifies unsupported files are skipped without losing valid files.
func TestIngestFolderReportsPartialFailures(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "brief.md")
	unsupportedPath := filepath.Join(root, "image.png")
	if err := os.WriteFile(validPath, []byte("folder brief\n"), 0600); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}
	if err := os.WriteFile(unsupportedPath, []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0xff}, 0600); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: root})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	if len(ingested) != 1 {
		t.Fatalf("expected 1 supported file event, got %d", len(ingested))
	}
	metadata := ingested[0].Metadata
	if metadata["filesystem_folder_skipped_count"] != "1" {
		t.Fatalf("filesystem_folder_skipped_count = %q, want 1", metadata["filesystem_folder_skipped_count"])
	}
	if !strings.Contains(metadata["filesystem_folder_first_error"], "image.png") {
		t.Fatalf("expected first skipped path to mention image.png, got %q", metadata["filesystem_folder_first_error"])
	}
}

// TestIngestExtractsDocumentFormats verifies docx, pptx, and pdf text extraction paths.
func TestIngestExtractsDocumentFormats(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		data   []byte
		want   string
		format string
	}{
		{name: "docx", file: "plan.docx", data: minimalDOCX(t), want: "Refund heading", format: "word_document"},
		{name: "pptx", file: "deck.pptx", data: minimalPPTX(t), want: "Slide 1: Delivery title", format: "presentation"},
		{name: "pdf", file: "brief.pdf", data: minimalPDF(), want: "PDF refund paragraph", format: "pdf"},
	}

	connector := filesystemsource.NewConnector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			if err := os.WriteFile(path, tt.data, 0600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
			if err != nil {
				t.Fatalf("ingest returned error: %v", err)
			}
			event := ingested[0]
			if !strings.Contains(event.Content, tt.want) {
				t.Fatalf("expected content to contain %q, got %q", tt.want, event.Content)
			}
			if event.Metadata["filesystem_format"] != tt.format {
				t.Fatalf("filesystem_format = %q, want %q", event.Metadata["filesystem_format"], tt.format)
			}
		})
	}
}

// TestIngestDelegatesSpreadsheetExtraction verifies filesystem spreadsheet ingestion uses cell-level facts.
func TestIngestDelegatesSpreadsheetExtraction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mapping.csv")
	if err := os.WriteFile(path, []byte("Name,Value\nStatus,Ready\n"), 0600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if !strings.Contains(event.Content, "Sheet1!A1=Name") || !strings.Contains(event.Content, "Sheet1!B2=Ready") {
		t.Fatalf("expected spreadsheet cell facts, got %q", event.Content)
	}
	if event.Metadata["filesystem_spreadsheet_cells"] != "4" {
		t.Fatalf("filesystem_spreadsheet_cells = %q, want 4", event.Metadata["filesystem_spreadsheet_cells"])
	}
}

// TestIngestExtractsXLSXSharedStringsAndFormulas verifies filesystem XLSX extraction preserves cell and formula facts.
func TestIngestExtractsXLSXSharedStringsAndFormulas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roadmap.xlsx")
	if err := os.WriteFile(path, minimalXLSX(t), 0600); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]

	for _, want := range []string{"Roadmap!A1=Status", "Roadmap!B1=Ready", "Roadmap!B2.formula=CONCAT(A1,\" ok\")"} {
		if !strings.Contains(event.Content, want) {
			t.Fatalf("expected content to contain %q, got %q", want, event.Content)
		}
	}
	if event.Metadata["filesystem_format"] != "spreadsheet" {
		t.Fatalf("filesystem_format = %q, want spreadsheet", event.Metadata["filesystem_format"])
	}
	if event.Metadata["filesystem_spreadsheet_sheets"] != "Roadmap" {
		t.Fatalf("filesystem_spreadsheet_sheets = %q, want Roadmap", event.Metadata["filesystem_spreadsheet_sheets"])
	}
	if event.Metadata["filesystem_spreadsheet_formulas"] != "1" {
		t.Fatalf("filesystem_spreadsheet_formulas = %q, want 1", event.Metadata["filesystem_spreadsheet_formulas"])
	}
}

// TestIngestSummarizesOpenAPIFile verifies OpenAPI specs are handled by filesystem ingestion.
func TestIngestSummarizesOpenAPIFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(path, []byte(sampleOpenAPISpec()), 0600); err != nil {
		t.Fatalf("write openapi fixture: %v", err)
	}

	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{URI: path})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if event.Metadata["filesystem_format"] != "openapi_spec" {
		t.Fatalf("filesystem_format = %q, want openapi_spec", event.Metadata["filesystem_format"])
	}
	if event.Metadata["openapi_title"] != "Payments API" || event.Metadata["openapi_operation_count"] != "2" {
		t.Fatalf("unexpected OpenAPI metadata: %#v", event.Metadata)
	}
	for _, want := range []string{"/paths/~1payments/get", "/paths/~1payments/post", "/components/schemas/Payment", "/components/schemas/Payment/properties/status/enum"} {
		if !strings.Contains(event.Metadata["openapi_source_locations"], want) {
			t.Fatalf("expected source locations to contain %q, got %q", want, event.Metadata["openapi_source_locations"])
		}
	}
	if event.Metadata[contracts.MetadataObjectType] != "file" {
		t.Fatalf("object_type = %q, want filesystem file identity", event.Metadata[contracts.MetadataObjectType])
	}
	if event.SourceID != "filesystem:file:"+path {
		t.Fatalf("source_id = %q, want filesystem source", event.SourceID)
	}
}

// TestIngestSummarizesInlineOpenAPIYAML verifies YAML specs remain filesystem file content.
func TestIngestSummarizesInlineOpenAPIYAML(t *testing.T) {
	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:     "payments.yaml",
		Content: sampleOpenAPIYAMLSpec(),
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if event.Metadata["filesystem_format"] != "openapi_spec" {
		t.Fatalf("filesystem_format = %q, want openapi_spec", event.Metadata["filesystem_format"])
	}
	if event.Metadata["openapi_title"] != "Payments YAML API" || event.Metadata["openapi_operation_count"] != "1" {
		t.Fatalf("unexpected OpenAPI YAML metadata: %#v", event.Metadata)
	}
	if event.Metadata[contracts.MetadataObjectType] != "file" {
		t.Fatalf("object_type = %q, want file", event.Metadata[contracts.MetadataObjectType])
	}
}

// TestIngestAppliesIncludeExcludeRules verifies explicit path filters are enforced before reading.
func TestIngestAppliesIncludeExcludeRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scratch.tmp")
	if err := os.WriteFile(path, []byte("ignore me"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	connector := filesystemsource.NewConnector()
	_, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: path,
		Metadata: map[string]string{
			"filesystem_exclude": "*.tmp",
		},
	})
	if err == nil {
		t.Fatal("expected excluded path error")
	}
	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		t.Fatalf("expected ConnectorError, got %T", err)
	}
	if connectorErr.Kind != contracts.ErrorKindPermanent || connectorErr.Retryable {
		t.Fatalf("expected non-retryable permanent error, got %#v", connectorErr)
	}
}

// TestIngestPreservesExplicitMetadataOverrides verifies caller-provided artifact IDs win over derived IDs.
func TestIngestPreservesExplicitMetadataOverrides(t *testing.T) {
	connector := filesystemsource.NewConnector()
	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI:     "requirements.csv",
		Content: "Name,Value\nRisk,High\n",
		Metadata: map[string]string{
			contracts.MetadataObjectType: "planning_sheet",
			contracts.MetadataObjectID:   "custom-workbook",
			events.MetadataSourceID:      "custom-source",
		},
	})
	if err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}
	event := ingested[0]
	if event.Metadata[contracts.MetadataObjectType] != "planning_sheet" {
		t.Fatalf("expected object type override, got %q", event.Metadata[contracts.MetadataObjectType])
	}
	if event.Metadata[contracts.MetadataObjectID] != "custom-workbook" {
		t.Fatalf("expected object id override, got %q", event.Metadata[contracts.MetadataObjectID])
	}
	if event.SourceID != "custom-source" {
		t.Fatalf("expected source id override, got %q", event.SourceID)
	}
}

func minimalDOCX(t *testing.T) []byte {
	t.Helper()
	return zipBytes(t, map[string]string{
		"word/document.xml": `<?xml version="1.0"?><w:document xmlns:w="urn:test"><w:body><w:p><w:r><w:t>Refund heading</w:t></w:r></w:p><w:p><w:r><w:t>Body paragraph</w:t></w:r></w:p></w:body></w:document>`,
	})
}

func minimalXLSX(t *testing.T) []byte {
	t.Helper()
	return zipBytes(t, map[string]string{
		"xl/workbook.xml":            `<?xml version="1.0"?><workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Roadmap" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0"?><Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<?xml version="1.0"?><sst><si><t>Status</t></si><si><t>Ready</t></si></sst>`,
		"xl/worksheets/sheet1.xml":   `<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row r="2"><c r="B2"><f>CONCAT(A1," ok")</f><v>ignored</v></c></row></sheetData></worksheet>`,
	})
}

func minimalPPTX(t *testing.T) []byte {
	t.Helper()
	return zipBytes(t, map[string]string{
		"ppt/slides/slide1.xml": `<?xml version="1.0"?><p:sld xmlns:p="urn:test" xmlns:a="urn:test"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Delivery title</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`,
	})
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func minimalPDF() []byte {
	return []byte("%PDF-1.4\n1 0 obj << /Length 57 >> stream\nBT /F1 12 Tf 72 720 Td (PDF refund paragraph) Tj ET\nendstream\nendobj\n%%EOF")
}

func sampleOpenAPISpec() string {
	return `{"openapi":"3.1.0","info":{"title":"Payments API","version":"2026.05"},"paths":{"/payments":{"get":{"responses":{"200":{"description":"ok"}}},"post":{"responses":{"201":{"description":"created"}}}}},"components":{"schemas":{"Payment":{"type":"object","properties":{"status":{"type":"string","enum":["pending","settled"]}}}}}}`
}

func sampleOpenAPIYAMLSpec() string {
	return strings.Join([]string{
		`openapi: "3.1.0"`,
		"info:",
		"  title: Payments YAML API",
		`  version: "2026.05"`,
		"paths:",
		"  /payments:",
		"    get:",
		"      responses:",
		`        "200":`,
		"          description: ok",
		"",
	}, "\n")
}
