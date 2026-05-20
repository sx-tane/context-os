package types

import "time"

type NormalizedDocument struct {
	ID           string            `json:"id"`
	Source       string            `json:"source"`
	SourceType   string            `json:"source_type"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Metadata     map[string]string `json:"metadata"`
	NormalizedAt time.Time         `json:"normalized_at"`
}

type Classification string

const (
	BusinessLogic Classification = "business_logic"
	APIDiscussion Classification = "api_discussion"
	PMORisk       Classification = "pmo_risk"
	FEConcern     Classification = "fe_concern"
	BEConcern     Classification = "be_concern"
	Blocker       Classification = "blocker"
	Decision      Classification = "decision"
	Unknown       Classification = "unknown"
)

type ClassifiedDocument struct {
	Document       NormalizedDocument `json:"document"`
	Classification Classification     `json:"classification"`
	Confidence     float64            `json:"confidence"`
}

type EntityType string

const (
	APIField    EntityType = "api_field"
	DBColumn    EntityType = "db_column"
	Enum        EntityType = "enum"
	Requirement EntityType = "requirement"
	Service     EntityType = "service"
	Dependency  EntityType = "dependency"
)

type Entity struct {
	ID       string            `json:"id"`
	Type     EntityType        `json:"type"`
	Name     string            `json:"name"`
	SourceID string            `json:"source_id"`
	Aliases  []string          `json:"aliases"`
	Metadata map[string]string `json:"metadata"`
}

type Relationship struct {
	ID       string            `json:"id"`
	FromID   string            `json:"from_id"`
	ToID     string            `json:"to_id"`
	Kind     string            `json:"kind"`
	Metadata map[string]string `json:"metadata"`
}

type Mismatch struct {
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	EntityIDs   []string `json:"entity_ids"`
	Severity    string   `json:"severity"`
	Recommended string   `json:"recommended"`
}
