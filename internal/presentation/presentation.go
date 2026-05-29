package presentation

import (
	"fmt"     // used to format the summary header line
	"strings" // used to join summary lines into a single string

	"context-os/domain/types" // Mismatch input type
)

// Role identifies the audience a summary is being rendered for.
type Role string

const (
	PMO               Role = "pmo"                // project management office view
	PresentationLayer Role = "presentation_layer" // presentation-layer (consumer) knowledge participant view
	ServiceLayer      Role = "service_layer"      // service-layer (producer) knowledge participant view
	QA                Role = "qa"                 // quality assurance view
	Architecture      Role = "architecture"       // system architecture view
)

// RenderSummary formats mismatch findings as a role-labelled plain-text summary.
// Each finding line carries its severity, summary, and—when present—the
// confidence, impact, and supporting evidence so the rendered view stays as
// explainable as the structured Mismatch it describes.
func RenderSummary(role Role, mismatches []types.Mismatch) string {
	if len(mismatches) == 0 {
		return fmt.Sprintf("%s view: no delivery mismatches detected", role) // fast path: clean state message
	}
	lines := []string{fmt.Sprintf("%s view: %d delivery mismatch(es) detected", role, len(mismatches))} // header line with count
	for _, mismatch := range mismatches {
		lines = append(lines, renderMismatch(mismatch)) // one bullet per mismatch with severity, confidence, impact, and evidence
	}
	return strings.Join(lines, "\n") // join all lines with newlines into a single output string
}

// renderMismatch formats a single mismatch as a bullet line, appending the
// explainability fields (confidence, impact, evidence) only when they are set.
func renderMismatch(mismatch types.Mismatch) string {
	line := fmt.Sprintf("- [%s] %s", mismatch.Severity, mismatch.Summary) // severity prefix and human-readable summary

	var qualifiers []string
	if mismatch.Confidence > 0 {
		qualifiers = append(qualifiers, fmt.Sprintf("confidence %.2f", mismatch.Confidence))
	}
	if strings.TrimSpace(mismatch.Impact) != "" {
		qualifiers = append(qualifiers, fmt.Sprintf("impact %s", mismatch.Impact))
	}
	if len(qualifiers) > 0 {
		line += fmt.Sprintf(" (%s)", strings.Join(qualifiers, ", "))
	}
	if len(mismatch.Evidence) > 0 {
		line += fmt.Sprintf(" evidence: %s", strings.Join(mismatch.Evidence, ", "))
	}
	return line
}
