package normalization

import (
	"strings" // used to trim whitespace from subject and content fields
	"time"    // used to stamp when normalization ran

	"context-os/domain/events" // Event type consumed as input
	"context-os/domain/types"  // NormalizedDocument type produced as output
)

// Normalize converts a raw pipeline event into a canonical NormalizedDocument.
func Normalize(event events.Event) types.NormalizedDocument {
	metadata := map[string]string{}          // create a fresh map so the event's map is never mutated
	for key, value := range event.Metadata { // copy every metadata entry from the event
		metadata[key] = value
	}
	return types.NormalizedDocument{
		ID:           event.ID,                        // carry the event's stable ID forward unchanged
		Source:       event.Source,                    // preserve which connector produced this document
		SourceType:   string(event.Type),              // convert the typed event kind to a plain string
		Title:        strings.TrimSpace(event.Subject), // remove leading/trailing whitespace from the subject
		Body:         strings.TrimSpace(event.Content), // remove leading/trailing whitespace from the content
		Metadata:     metadata,                        // attach the copied metadata
		NormalizedAt: time.Now().UTC(),                // record when this normalization happened
	}
}
