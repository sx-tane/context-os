package presentation

import (
	"context-os/domain/types"
	"sort"
	"strings"
)

// collectMismatchIDs returns sorted mismatch IDs for stable traces and responses.
func collectMismatchIDs(mismatches []types.Mismatch) []string {
	ids := make([]string, 0, len(mismatches))
	for _, mismatch := range mismatches {
		ids = append(ids, mismatch.ID)
	}
	sort.Strings(ids)
	return ids
}

// collectRecommendations returns unique non-empty recommendations in encounter order.
func collectRecommendations(mismatches []types.Mismatch) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, mismatch := range mismatches {
		recommended := strings.TrimSpace(mismatch.Recommended)
		if recommended == "" {
			continue
		}
		if _, ok := seen[recommended]; ok {
			continue
		}
		seen[recommended] = struct{}{}
		out = append(out, recommended)
	}
	return out
}

// pluralSuffix returns the English plural suffix for count-sensitive messages.
func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// severityCount counts mismatches by normalized severity with medium as the default.
func severityCount(mismatches []types.Mismatch) map[string]int {
	out := map[string]int{"low": 0, "medium": 0, "high": 0}
	for _, mismatch := range mismatches {
		severity := strings.ToLower(strings.TrimSpace(mismatch.Severity))
		if severity == "" {
			severity = "medium"
		}
		out[severity]++
	}
	return out
}
