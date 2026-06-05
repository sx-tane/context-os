package response

import "context-os/domain/types"

// PresentationFindings is the JSON payload returned by POST /presentation/findings.
type PresentationFindings struct {
	Connector            string            `json:"connector"`
	URI                  string            `json:"uri"`
	Role                 string            `json:"role"`
	TraceID              string            `json:"trace_id"`
	Summary              string            `json:"summary"`
	EventCount           int               `json:"event_count"`
	EntityCount          int               `json:"entity_count,omitempty"`
	RelationshipCount    int               `json:"relationship_count,omitempty"`
	MismatchCount        int               `json:"mismatch_count"`
	ReviewCandidateCount int               `json:"review_candidate_count"`
	SeverityCount        map[string]int    `json:"severity_count"`
	MismatchIDs          []string          `json:"mismatch_ids"`
	Mismatches           []types.Mismatch  `json:"mismatches"`
	ReviewCandidates     []types.Mismatch  `json:"review_candidates,omitempty"`
	Views                RoleViews         `json:"views"`
	PMO                  PMOSummary        `json:"pmo"`
	Execution            ExecutionEvidence `json:"execution"`
}

// RoleSummaryView carries one role-specific summary contract.
type RoleSummaryView struct {
	Role         string   `json:"role"`
	Summary      string   `json:"summary"`
	MismatchIDs  []string `json:"mismatch_ids"`
	NextActions  []string `json:"next_actions"`
	FindingCount int      `json:"finding_count"`
}

// RoleViews provides explicit response shapes for all supported roles.
type RoleViews struct {
	PMO               RoleSummaryView `json:"pmo"`
	PresentationLayer RoleSummaryView `json:"presentation_layer"`
	ServiceLayer      RoleSummaryView `json:"service_layer"`
	QA                RoleSummaryView `json:"qa"`
	Architecture      RoleSummaryView `json:"architecture"`
}

// PMOSummary exposes a PMO-specific view model that separates facts, risk, impact,
// confidence, evidence, and recommended decisions.
type PMOSummary struct {
	Facts                []string            `json:"facts"`
	Risks                []string            `json:"risks"`
	Impacts              []string            `json:"impacts"`
	Confidence           map[string]float64  `json:"confidence"`
	Evidence             map[string][]string `json:"evidence"`
	RecommendedDecisions []string            `json:"recommended_decisions"`
}

// ExecutionEvidence reports hidden executor output as assistive evidence.
type ExecutionEvidence struct {
	Enabled   bool              `json:"enabled"`
	Assistive bool              `json:"assistive"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata"`
	Error     string            `json:"error,omitempty"`
}
