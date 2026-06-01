package extraction

import (
	"fmt"     // used to build deterministic entity IDs
	"regexp"  // used to match token patterns in the document body
	"strings" // used for trimming and lowercasing names

	"context-os/domain/contracts" // provenance metadata keys copied onto extracted entities
	"context-os/domain/events"    // source artifact metadata keys copied onto extracted entities
	"context-os/domain/types"     // ClassifiedDocument input and Entity output
)

// Extraction method labels record how each entity was produced so downstream stages can weigh provenance.
const (
	MethodRegexToken  = "regex_token"  // matched by the generic identifier regex
	MethodOpenAPI     = "openapi"      // parsed from an OpenAPI/Swagger specification
	MethodSpreadsheet = "spreadsheet"  // parsed from spreadsheet cell content
	MethodJiraField   = "jira_field"   // parsed from a Jira issue payload
	MethodGitHubField = "github_field" // parsed from a GitHub issue or pull request payload
)

// Connector metadata keys used to route a document to a structured extractor.
const (
	metadataConnector     = "connector"         // connector name set by the MCP base connector
	metadataObjectType    = "object_type"       // artifact kind attached by connectors
	metadataFilesystemFmt = "filesystem_format" // filesystem connector format tag
	formatOpenAPISpec     = "openapi_spec"      // filesystem_format value for OpenAPI specs
	formatSpreadsheet     = "spreadsheet"       // filesystem_format value for CSV/XLSX
	connectorJira         = "jira"              // connector name for Jira
	connectorGitHub       = "github"            // connector name for GitHub
	regexTokenConfidence  = 0.5                 // confidence assigned to generic regex matches
	structuredConfidence  = 0.9                 // confidence assigned to structured extractions
)

// tokenPattern matches identifiers that look like named concepts (e.g. refundStatus, UserID, paymentFlag).
var tokenPattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*(?:Status|State|ID|Id|Type|Flag|Field|Column)?`)

// Extract turns a classified document into structured entities, routing to a structured
// extractor when the source metadata identifies one and falling back to generic token
// extraction otherwise.
func Extract(doc types.ClassifiedDocument) []types.Entity {
	meta := doc.Document.Metadata
	switch {
	case meta[metadataFilesystemFmt] == formatOpenAPISpec:
		if entities := extractOpenAPI(doc); len(entities) > 0 {
			return entities
		}
	case meta[metadataFilesystemFmt] == formatSpreadsheet:
		if entities := extractSpreadsheet(doc); len(entities) > 0 {
			return entities
		}
	case meta[metadataConnector] == connectorJira:
		if entities := extractJira(doc); len(entities) > 0 {
			return entities
		}
	case meta[metadataConnector] == connectorGitHub:
		if entities := extractGitHub(doc); len(entities) > 0 {
			return entities
		}
	}
	return extractTokens(doc) // deterministic fallback for unstructured or unparseable content
}

// extractTokens pulls named entity tokens from a classified document's body text.
func extractTokens(doc types.ClassifiedDocument) []types.Entity {
	matches := tokenPattern.FindAllStringIndex(doc.Document.Body, -1) // find token candidates with offsets
	seen := map[string]bool{}                                         // track which canonical keys have already been added
	entities := make([]types.Entity, 0, len(matches))                 // pre-allocate with a reasonable capacity
	for _, loc := range matches {                                     // process each regex match in order
		raw := doc.Document.Body[loc[0]:loc[1]] // original matched text including any casing
		name := strings.TrimSpace(raw)          // clean surrounding whitespace from the token
		key := strings.ToLower(name)            // normalise to lowercase for deduplication
		if len(name) < 3 || seen[key] {         // skip tokens that are too short or already captured
			continue
		}
		seen[key] = true // mark this key so future duplicates are skipped
		span := types.SourceSpan{Field: "body", Start: loc[0], End: loc[1]}
		entities = append(entities, newEntity(doc, key, name, raw, inferType(name, doc.Classification), MethodRegexToken, regexTokenConfidence, span))
	}
	return entities // return the deduplicated list of extracted entities
}

// newEntity builds an entity with consistent provenance, confidence, and span fields.
func newEntity(doc types.ClassifiedDocument, key, name, raw string, entityType types.EntityType, method string, confidence float64, spans ...types.SourceSpan) types.Entity {
	return types.Entity{
		ID:               fmt.Sprintf("%s:%s", doc.Document.ID, key), // combine document ID and key for a stable entity ID
		Type:             entityType,                                 // determine what kind of concept this token represents
		Name:             name,                                       // preserve the original casing from the text
		RawMention:       raw,                                        // keep the exact source text before normalization
		SourceID:         doc.Document.ID,                            // link back to the document it came from
		Confidence:       confidence,                                 // how certain this extraction is
		ExtractionMethod: method,                                     // how the entity was produced
		Spans:            spans,                                      // offsets or pointers locating the mention
		Metadata:         entityMetadata(doc),                        // carry classification and source evidence forward
	}
}

func entityMetadata(doc types.ClassifiedDocument) map[string]string {
	metadata := map[string]string{"classification": string(doc.Classification)}
	if sourceURI := strings.TrimSpace(doc.Document.Metadata[contracts.MetadataSourceURI]); sourceURI != "" {
		metadata[contracts.MetadataSourceURI] = sourceURI
	}
	if sourceID := strings.TrimSpace(doc.Document.Metadata[events.MetadataSourceID]); sourceID != "" {
		metadata[events.MetadataSourceID] = sourceID
	}
	return metadata
}

// inferType maps an entity name and its document classification to the most likely EntityType.
func inferType(name string, classification types.Classification) types.EntityType {
	lower := strings.ToLower(name) // lowercase once for all comparisons below
	switch {
	case isLikelyAPIFieldName(name, lower):
		return types.APIField // status/state/field names are typically API schema fields
	case strings.Contains(lower, "column") || strings.Contains(lower, "database"):
		return types.DBColumn // column or database names map to DB concepts
	case strings.Contains(lower, "type") || strings.Contains(lower, "flag"):
		return types.Enum // type/flag names are usually enumerated values
	case classification == types.BusinessLogic:
		return types.Requirement // tokens from business logic documents are treated as requirements
	case classification == types.APIDiscussion:
		return types.Service // tokens from API discussions are treated as service names
	default:
		return types.Dependency // everything else is a generic dependency
	}
}

// isLikelyAPIFieldName prevents generic plain-language words (for example,
// "status") from being treated as schema fields when token fallback is used.
func isLikelyAPIFieldName(name, lower string) bool {
	if strings.Contains(lower, "field") {
		return true
	}
	if !strings.Contains(lower, "status") && !strings.Contains(lower, "state") {
		return false
	}
	if strings.Contains(name, "_") {
		return true
	}
	return name != lower
}
