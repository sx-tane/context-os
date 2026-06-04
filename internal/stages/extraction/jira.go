package extraction

import (
	"encoding/json" // used to locate the nested fields object

	"context-os/domain/types" // Entity output type
)

// extractJira parses a Jira issue payload and emits structured entities for each scalar field
// under the issue's "fields" object. When the content is not valid Jira JSON it returns nil so the
// dispatcher falls back to generic token extraction.
func extractJira(doc types.ClassifiedDocument) []types.Entity {
	object, ok := decodeObject(doc.Document.Body)
	if !ok {
		return nil
	}
	rawFields, ok := object["fields"]
	if !ok {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawFields, &fields); err != nil {
		return nil
	}
	return scalarFieldEntities(doc, fields, "/fields", MethodJiraField)
}
