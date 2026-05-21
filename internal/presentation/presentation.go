package presentation

import (
	"fmt"     // used to format the summary header line
	"strings" // used to join summary lines into a single string

	"context-os/domain/types" // Mismatch input type
)

// Role identifies the audience a summary is being rendered for.
type Role string

const (
	PMO              Role = "pmo"               // project management office view
	PresentationLayer Role = "presentation_layer" // presentation-layer (consumer) knowledge participant view
	ServiceLayer      Role = "service_layer"      // service-layer (producer) knowledge participant view
	QA               Role = "qa"                // quality assurance view
	Architecture     Role = "architecture"       // system architecture view
)

// RenderSummary formats mismatch findings as a role-labelled plain-text summary.
func RenderSummary(role Role, mismatches []types.Mismatch) string {
	if len(mismatches) == 0 {
		return fmt.Sprintf("%s view: no delivery mismatches detected", role) // fast path: clean state message
	}
	lines := []string{fmt.Sprintf("%s view: %d delivery mismatch(es) detected", role, len(mismatches))} // header line with count
	for _, mismatch := range mismatches {
		lines = append(lines, fmt.Sprintf("- [%s] %s", mismatch.Severity, mismatch.Summary)) // one bullet per mismatch with severity prefix
	}
	return strings.Join(lines, "\n") // join all lines with newlines into a single output string
}
