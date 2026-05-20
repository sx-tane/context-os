package presentation

import (
	"fmt"
	"strings"

	"github.com/sx-tane/context-os/shared/types"
)

type Role string

const (
	PMO          Role = "pmo"
	Frontend     Role = "frontend"
	Backend      Role = "backend"
	QA           Role = "qa"
	Architecture Role = "architecture"
)

func RenderSummary(role Role, mismatches []types.Mismatch) string {
	if len(mismatches) == 0 {
		return fmt.Sprintf("%s view: no delivery mismatches detected", role)
	}
	lines := []string{fmt.Sprintf("%s view: %d delivery mismatch(es) detected", role, len(mismatches))}
	for _, mismatch := range mismatches {
		lines = append(lines, fmt.Sprintf("- [%s] %s", mismatch.Severity, mismatch.Summary))
	}
	return strings.Join(lines, "\n")
}
