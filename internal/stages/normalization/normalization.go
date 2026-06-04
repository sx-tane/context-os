package normalization

import (
	"crypto/sha256" // used to derive a deterministic content hash for replay detection
	"encoding/hex"  // used to render the content hash as a stable string
	"strings"       // used to trim whitespace from subject and content fields
	"time"          // used to stamp when normalization ran
	"unicode/utf8"  // used to measure span lengths in runes

	"context-os/domain/events" // Event type consumed as input
	"context-os/domain/types"  // NormalizedDocument type produced as output
)

// RuleVersion identifies the normalization rule set so downstream replays can detect transform changes.
const RuleVersion = "normalization/v1"

// metadataContentHash is the connector-provided content hash reused when present to keep ingest and normalization aligned.
const metadataContentHash = "filesystem_content_hash"

// Normalize converts a raw pipeline event into a canonical NormalizedDocument.
func Normalize(event events.Event) types.NormalizedDocument {
	metadata := map[string]string{}          // create a fresh map so the event's map is never mutated
	for key, value := range event.Metadata { // copy every metadata entry from the event
		metadata[key] = value
	}

	title := strings.TrimSpace(event.Subject) // remove leading/trailing whitespace from the subject
	body := strings.TrimSpace(event.Content)  // remove leading/trailing whitespace from the content

	return types.NormalizedDocument{
		ID:            event.ID,                        // carry the event's stable ID forward unchanged
		Source:        event.Source,                    // preserve which connector produced this document
		SourceType:    string(event.Type),              // convert the typed event kind to a plain string
		Title:         title,                           // canonical, whitespace-trimmed title
		Body:          body,                            // canonical, whitespace-trimmed body
		Metadata:      metadata,                        // attach the copied metadata
		ContentHash:   contentHash(event, title, body), // deterministic hash for replay detection
		SchemaVersion: event.SchemaVersion,             // carry the event schema version for migration safety
		RuleVersion:   RuleVersion,                     // record which normalization rules ran
		Spans:         spans(title, body),              // offsets locating canonical text back in the source
		NormalizedAt:  time.Now().UTC(),                // record when this normalization happened
	}
}

// contentHash returns a deterministic hash of the canonical text, reusing the connector hash when supplied.
func contentHash(event events.Event, title, body string) string {
	if existing := strings.TrimSpace(event.Metadata[metadataContentHash]); existing != "" {
		return existing // trust the connector's content hash so ingest and normalization stay consistent
	}
	sum := sha256.Sum256([]byte(title + "\n" + body)) // hash the canonical title and body together
	return hex.EncodeToString(sum[:])
}

// spans records rune offsets for the non-empty canonical fields so findings can be traced back to source text.
func spans(title, body string) []types.SourceSpan {
	result := []types.SourceSpan{}
	if title != "" {
		result = append(result, types.SourceSpan{Field: "title", Start: 0, End: utf8.RuneCountInString(title)})
	}
	if body != "" {
		result = append(result, types.SourceSpan{Field: "body", Start: 0, End: utf8.RuneCountInString(body)})
	}
	return result
}
