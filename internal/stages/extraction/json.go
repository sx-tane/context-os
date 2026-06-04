package extraction

import (
	"encoding/json" // used to decode connector JSON payloads
	"sort"          // used to keep field iteration deterministic
	"strings"       // used to clean field names

	"context-os/domain/types" // Entity output type
)

// scalarFieldEntities decodes the object found at the given JSON pointer prefix and emits one
// entity per scalar field, using the field name as the entity name and the pointer as its span
// path. It returns nil when the content is not valid JSON so callers can fall back to token
// extraction. fields is the decoded object (already located by the caller).
func scalarFieldEntities(doc types.ClassifiedDocument, fields map[string]json.RawMessage, pointerPrefix, method string) []types.Entity {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic ordering independent of map iteration

	seen := map[string]bool{}
	entities := []types.Entity{}
	for _, name := range names {
		if !isScalarJSON(fields[name]) {
			continue // skip nested objects/arrays; this stage extracts named fields only
		}
		clean := strings.TrimSpace(name)
		canonical := strings.ToLower(clean)
		if len(clean) < 2 || seen[canonical] {
			continue
		}
		seen[canonical] = true
		span := types.SourceSpan{Field: "body", Path: pointerPrefix + "/" + clean}
		entities = append(entities, newEntity(doc, canonical, clean, clean, inferType(clean, doc.Classification), method, structuredConfidence, span))
	}
	return entities
}

// isScalarJSON reports whether raw is a JSON string, number, or boolean (not object, array, or null).
func isScalarJSON(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	switch trimmed[0] {
	case '{', '[':
		return false
	default:
		return true
	}
}

// decodeObject unmarshals raw JSON content into a string-keyed object, returning ok=false on failure.
func decodeObject(content string) (map[string]json.RawMessage, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || trimmed[0] != '{' {
		return nil, false
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &object); err != nil {
		return nil, false
	}
	return object, true
}
