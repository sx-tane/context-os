package extraction

import (
	"encoding/json" // used to decode the connector's JSON pointer arrays
	"strings"       // used to split and clean JSON pointer segments

	"context-os/domain/types" // Entity output type
)

// OpenAPI metadata keys carrying JSON pointer arrays attached by the filesystem connector.
const (
	metadataOpenAPIEndpointPointers = "openapi_endpoint_pointers" // pointers to operations/endpoints
	metadataOpenAPISchemaPointers   = "openapi_schema_pointers"   // pointers to component schemas
	metadataOpenAPIEnumPointers     = "openapi_enum_pointers"     // pointers to enum definitions
)

// extractOpenAPI turns the OpenAPI pointer metadata attached by the filesystem connector into
// structured entities for endpoints, schemas, and enum values. Each entity keeps the JSON pointer
// as its source span path so findings remain traceable to the spec location.
func extractOpenAPI(doc types.ClassifiedDocument) []types.Entity {
	meta := doc.Document.Metadata
	seen := map[string]bool{}
	entities := []types.Entity{}

	collect := func(key string, entityType types.EntityType) {
		for _, pointer := range decodePointers(meta[key]) {
			name := pointerName(pointer)
			canonical := strings.ToLower(name)
			if len(name) < 2 || seen[canonical] {
				continue
			}
			seen[canonical] = true
			span := types.SourceSpan{Field: "body", Path: pointer}
			entities = append(entities, newEntity(doc, canonical, name, pointer, entityType, MethodOpenAPI, structuredConfidence, span))
		}
	}

	collect(metadataOpenAPIEndpointPointers, types.Service)
	collect(metadataOpenAPISchemaPointers, types.APIField)
	collect(metadataOpenAPIEnumPointers, types.Enum)

	return entities
}

// decodePointers parses a JSON array of pointer strings, tolerating empty or malformed metadata.
func decodePointers(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var pointers []string
	if err := json.Unmarshal([]byte(raw), &pointers); err != nil {
		return nil
	}
	return pointers
}

// pointerName returns the decoded last segment of a JSON pointer as a human-facing entity name.
func pointerName(pointer string) string {
	segments := strings.Split(strings.Trim(pointer, "/"), "/")
	last := segments[len(segments)-1]
	last = strings.ReplaceAll(last, "~1", "/") // JSON pointer escape for '/'
	last = strings.ReplaceAll(last, "~0", "~") // JSON pointer escape for '~'
	return strings.TrimSpace(last)
}
