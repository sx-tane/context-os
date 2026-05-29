package extraction

import (
	"fmt"     // used to build deterministic entity IDs
	"regexp"  // used to match token patterns in the document body
	"strings" // used for trimming and lowercasing names

	"context-os/domain/contracts" // provenance metadata keys copied onto extracted entities
	"context-os/domain/events"    // source artifact metadata keys copied onto extracted entities
	"context-os/domain/types"     // ClassifiedDocument input and Entity output
)

// tokenPattern matches identifiers that look like named concepts (e.g. refundStatus, UserID, paymentFlag).
var tokenPattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*(?:Status|State|ID|Id|Type|Flag|Field|Column)?`)

// Extract pulls named entity tokens from a classified document's body text.
func Extract(doc types.ClassifiedDocument) []types.Entity {
	matches := tokenPattern.FindAllString(doc.Document.Body, -1) // find all token candidates in the body
	seen := map[string]bool{}                                    // track which canonical keys have already been added
	entities := make([]types.Entity, 0, len(matches))            // pre-allocate with a reasonable capacity
	for _, match := range matches {                              // process each regex match in order
		name := strings.TrimSpace(match) // clean surrounding whitespace from the token
		key := strings.ToLower(name)     // normalise to lowercase for deduplication
		if len(name) < 3 || seen[key] {  // skip tokens that are too short or already captured
			continue
		}
		seen[key] = true // mark this key so future duplicates are skipped
		entities = append(entities, types.Entity{
			ID:       fmt.Sprintf("%s:%s", doc.Document.ID, key), // combine document ID and key for a stable entity ID
			Type:     inferType(name, doc.Classification),        // determine what kind of concept this token represents
			Name:     name,                                       // preserve the original casing from the text
			SourceID: doc.Document.ID,                            // link back to the document it came from
			Metadata: entityMetadata(doc),                        // carry classification and source evidence forward
		})
	}
	return entities // return the deduplicated list of extracted entities
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
	case strings.Contains(lower, "field") || strings.Contains(lower, "status") || strings.Contains(lower, "state"):
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
