package extraction

import "context-os/domain/types" // Entity output type

// extractGitHub parses a GitHub issue or pull request payload and emits structured entities for
// each top-level scalar field. When the content is not valid GitHub JSON it returns nil so the
// dispatcher falls back to generic token extraction.
func extractGitHub(doc types.ClassifiedDocument) []types.Entity {
	object, ok := decodeObject(doc.Document.Body)
	if !ok {
		return nil
	}
	return scalarFieldEntities(doc, object, "", MethodGitHubField)
}
