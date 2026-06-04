package extraction

import (
	"strings" // used to parse cell reference lines from spreadsheet content

	"context-os/domain/types" // Entity output type
)

// extractSpreadsheet turns the filesystem connector's "Sheet!Cell=value" content lines into
// structured entities. Each cell value becomes a candidate entity whose source span path is the
// cell reference, preserving workbook provenance for PMO and requirement findings.
func extractSpreadsheet(doc types.ClassifiedDocument) []types.Entity {
	seen := map[string]bool{}
	entities := []types.Entity{}

	for _, line := range strings.Split(doc.Document.Body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ref, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		ref = strings.TrimSpace(ref)
		if strings.HasSuffix(ref, ".formula") {
			continue // formula lines describe derivation, not a named concept
		}
		value = strings.TrimSpace(value)
		canonical := strings.ToLower(value)
		if len(value) < 3 || seen[canonical] {
			continue
		}
		seen[canonical] = true
		span := types.SourceSpan{Field: "body", Path: ref}
		entities = append(entities, newEntity(doc, canonical, value, value, inferType(value, doc.Classification), MethodSpreadsheet, structuredConfidence, span))
	}

	return entities
}
