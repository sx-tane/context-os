package presentation

import (
	"context-os/apps/api/response"
	"context-os/domain/types"
	stagepresentation "context-os/internal/stages/presentation"
	"fmt"
	"strings"
)

// parseRole normalizes API role input to a supported presentation role.
func parseRole(value string) stagepresentation.Role {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(stagepresentation.PresentationLayer):
		return stagepresentation.PresentationLayer
	case string(stagepresentation.ServiceLayer):
		return stagepresentation.ServiceLayer
	case string(stagepresentation.QA):
		return stagepresentation.QA
	case string(stagepresentation.Architecture):
		return stagepresentation.Architecture
	default:
		return stagepresentation.PMO
	}
}

// buildRoleViews builds every role-specific summary view from actionable mismatches.
func buildRoleViews(mismatches []types.Mismatch) response.RoleViews {
	return response.RoleViews{
		PMO:               buildRoleSummary(stagepresentation.PMO, mismatches),
		PresentationLayer: buildRoleSummary(stagepresentation.PresentationLayer, mismatches),
		ServiceLayer:      buildRoleSummary(stagepresentation.ServiceLayer, mismatches),
		QA:                buildRoleSummary(stagepresentation.QA, mismatches),
		Architecture:      buildRoleSummary(stagepresentation.Architecture, mismatches),
	}
}

// splitPresentationFindings separates actionable mismatches from dependency review candidates.
func splitPresentationFindings(mismatches []types.Mismatch) ([]types.Mismatch, []types.Mismatch) {
	actionable := make([]types.Mismatch, 0, len(mismatches))
	reviewCandidates := []types.Mismatch{}
	for _, mismatch := range mismatches {
		if isReviewCandidate(mismatch) {
			reviewCandidates = append(reviewCandidates, normalizeReviewCandidate(mismatch))
			continue
		}
		actionable = append(actionable, mismatch)
	}
	return actionable, reviewCandidates
}

// isReviewCandidate identifies low-priority dependency findings that should stay out of top issues.
func isReviewCandidate(mismatch types.Mismatch) bool {
	switch strings.ToLower(strings.TrimSpace(mismatch.Type)) {
	case "dependency_risk", "dependency_review":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mismatch.ID)), "dependency_risk:")
	}
}

// normalizeReviewCandidate downgrades dependency risk findings into review candidates.
func normalizeReviewCandidate(mismatch types.Mismatch) types.Mismatch {
	mismatch.Type = "dependency_review"
	mismatch.Severity = "low"
	mismatch.Impact = "low"
	if strings.TrimSpace(mismatch.Recommended) == "" {
		mismatch.Recommended = "Review this dependency when the affected service becomes delivery-critical."
	}
	return mismatch
}

// renderFindingsSummary renders the selected role summary and notes hidden review candidates.
func renderFindingsSummary(role stagepresentation.Role, actionable, reviewCandidates []types.Mismatch) string {
	base := stagepresentation.RenderSummary(role, actionable)
	if len(reviewCandidates) == 0 {
		return base
	}
	return fmt.Sprintf("%s Review candidates: %d dependency item%s kept out of Top issues.", base, len(reviewCandidates), pluralSuffix(len(reviewCandidates)))
}

// buildRoleSummary creates one role summary view with IDs and next actions.
func buildRoleSummary(role stagepresentation.Role, mismatches []types.Mismatch) response.RoleSummaryView {
	return response.RoleSummaryView{
		Role:         string(role),
		Summary:      stagepresentation.RenderSummary(role, mismatches),
		MismatchIDs:  collectMismatchIDs(mismatches),
		NextActions:  collectRecommendations(mismatches),
		FindingCount: len(mismatches),
	}
}

// buildPMOSummary creates the PMO decision model from actionable mismatches.
func buildPMOSummary(mismatches []types.Mismatch) response.PMOSummary {
	summary := response.PMOSummary{
		Facts:                make([]string, 0, len(mismatches)),
		Risks:                []string{},
		Impacts:              []string{},
		Confidence:           map[string]float64{},
		Evidence:             map[string][]string{},
		RecommendedDecisions: []string{},
	}
	seenImpact := map[string]struct{}{}
	seenDecision := map[string]struct{}{}
	for _, mismatch := range mismatches {
		summary.Facts = append(summary.Facts, fmt.Sprintf("%s: %s", mismatch.ID, mismatch.Summary))
		if strings.EqualFold(mismatch.Severity, "high") {
			summary.Risks = append(summary.Risks, fmt.Sprintf("%s (%s)", mismatch.Summary, mismatch.ID))
		}
		if impact := strings.TrimSpace(mismatch.Impact); impact != "" {
			if _, ok := seenImpact[impact]; !ok {
				seenImpact[impact] = struct{}{}
				summary.Impacts = append(summary.Impacts, impact)
			}
		}
		summary.Confidence[mismatch.ID] = mismatch.Confidence
		summary.Evidence[mismatch.ID] = append([]string(nil), mismatch.Evidence...)
		if decision := strings.TrimSpace(mismatch.Recommended); decision != "" {
			if _, ok := seenDecision[decision]; !ok {
				seenDecision[decision] = struct{}{}
				summary.RecommendedDecisions = append(summary.RecommendedDecisions, decision)
			}
		}
	}
	return summary
}
