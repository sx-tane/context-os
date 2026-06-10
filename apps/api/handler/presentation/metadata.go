package presentation

import (
	"strings"
)

// cloneMetadata copies non-empty request metadata values before connector-specific mutation.
func cloneMetadata(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[key] = trimmed
	}
	return out
}

// setIfNotEmpty stores a metadata value only when the input is non-blank.
func setIfNotEmpty(metadata map[string]string, key, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	metadata[key] = trimmed
}
