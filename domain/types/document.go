package types

import "time" // used for the normalization timestamp on NormalizedDocument

// NormalizedDocument is the canonical form every source artifact is converted into.
type NormalizedDocument struct {
	ID           string            `json:"id"`            // stable identifier inherited from the originating event
	Source       string            `json:"source"`        // connector name that produced this document
	SourceType   string            `json:"source_type"`   // event type string, e.g. "document.ingested"
	Title        string            `json:"title"`         // trimmed subject line of the artifact
	Body         string            `json:"body"`          // trimmed content body of the artifact
	Metadata     map[string]string `json:"metadata"`      // key-value pairs carried from the source event
	NormalizedAt time.Time         `json:"normalized_at"` // UTC timestamp of when normalization ran
}

// Classification categorises a document by its delivery signal type.
type Classification string

const (
	BusinessLogic Classification = "business_logic"  // document describes business rules or requirements
	APIDiscussion Classification = "api_discussion"   // document discusses API contracts or endpoints
	PMORisk       Classification = "pmo_risk"         // document flags project risk or delay
	ConsumerConcern Classification = "consumer_concern" // document is primarily a presentation-layer (consumer) concern
	ProducerConcern Classification = "producer_concern" // document is primarily a service-layer (producer) concern
	Blocker       Classification = "blocker"           // document signals a blocking issue
	Decision      Classification = "decision"          // document records a team decision
	Unknown       Classification = "unknown"           // document did not match any known signal pattern
)

// ClassifiedDocument pairs a normalized document with its detected classification and confidence.
type ClassifiedDocument struct {
	Document       NormalizedDocument `json:"document"`       // the normalized artifact being classified
	Classification Classification     `json:"classification"` // the best matching signal category
	Confidence     float64            `json:"confidence"`     // 0.0–1.0 score for how certain the classification is
}

// EntityType describes what kind of concept an extracted entity represents.
type EntityType string

const (
	APIField    EntityType = "api_field"   // a field in an API request or response schema
	DBColumn    EntityType = "db_column"   // a column in a database table
	Enum        EntityType = "enum"        // an enumerated value or status flag
	Requirement EntityType = "requirement" // a stated business or functional requirement
	Service     EntityType = "service"     // a named service, system, or component
	Dependency  EntityType = "dependency"  // a general dependency not fitting other types
)

// Entity represents a named concept extracted from a document.
type Entity struct {
	ID       string            `json:"id"`        // unique identifier combining source and canonical key
	Type     EntityType        `json:"type"`      // what kind of concept this entity is
	Name     string            `json:"name"`      // the surface form of the entity as it appeared in the text
	SourceID string            `json:"source_id"` // ID of the document this entity was extracted from
	Aliases  []string          `json:"aliases"`   // all known name variants merged during identity resolution
	Metadata map[string]string `json:"metadata"`  // additional context attached during extraction
}

// Relationship describes a directed link between two entities.
type Relationship struct {
	ID       string            `json:"id"`        // deterministic ID built from the two entity IDs
	FromID   string            `json:"from_id"`   // ID of the source entity
	ToID     string            `json:"to_id"`     // ID of the target entity
	Kind     string            `json:"kind"`      // label describing how the two entities relate
	Metadata map[string]string `json:"metadata"`  // extra context attached by the relationship builder
}

// Mismatch describes a detected delivery misalignment between teams or artifacts.
type Mismatch struct {
	ID          string   `json:"id"`          // unique identifier for this mismatch finding
	Summary     string   `json:"summary"`     // human-readable description of what was found
	EntityIDs   []string `json:"entity_ids"`  // IDs of the entities involved in the mismatch
	Severity    string   `json:"severity"`    // impact level: low, medium, or high
	Recommended string   `json:"recommended"` // suggested action for the team to resolve this
}
