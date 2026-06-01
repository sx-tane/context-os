package types

import "time" // used for the normalization timestamp on NormalizedDocument

// SourceSpan locates a slice of normalized text back in the artifact it came from.
type SourceSpan struct {
	Field string `json:"field"` // which normalized field the span refers to, e.g. "title" or "body"
	Start int    `json:"start"` // inclusive start offset (in runes) within the field
	End   int    `json:"end"`   // exclusive end offset (in runes) within the field
	Path  string `json:"path"`  // optional structured pointer into the source, e.g. a JSON pointer or cell ref
}

// NormalizedDocument is the canonical form every source artifact is converted into.
type NormalizedDocument struct {
	ID            string            `json:"id"`             // stable identifier inherited from the originating event
	Source        string            `json:"source"`         // connector name that produced this document
	SourceType    string            `json:"source_type"`    // event type string, e.g. "document.ingested"
	Title         string            `json:"title"`          // trimmed subject line of the artifact
	Body          string            `json:"body"`           // trimmed content body of the artifact
	Metadata      map[string]string `json:"metadata"`       // key-value pairs carried from the source event
	ContentHash   string            `json:"content_hash"`   // deterministic hash of the canonical text for replay detection
	SchemaVersion string            `json:"schema_version"` // event schema version this document was normalized under
	RuleVersion   string            `json:"rule_version"`   // version of the normalization rules that produced this document
	Spans         []SourceSpan      `json:"spans"`          // offsets locating canonical text back in the source artifact
	NormalizedAt  time.Time         `json:"normalized_at"`  // UTC timestamp of when normalization ran
}

// Classification categorises a document by its delivery signal type.
type Classification string

const (
	BusinessLogic   Classification = "business_logic"   // document describes business rules or requirements
	APIDiscussion   Classification = "api_discussion"   // document discusses API contracts or endpoints
	PMORisk         Classification = "pmo_risk"         // document flags project risk or delay
	ConsumerConcern Classification = "consumer_concern" // document is primarily a presentation-layer (consumer) concern
	ProducerConcern Classification = "producer_concern" // document is primarily a service-layer (producer) concern
	Blocker         Classification = "blocker"          // document signals a blocking issue
	Decision        Classification = "decision"         // document records a team decision
	Unknown         Classification = "unknown"          // document did not match any known signal pattern
)

// ScoredLabel is one delivery-signal category a document matched, with its supporting evidence.
type ScoredLabel struct {
	Classification Classification `json:"classification"` // the matched signal category
	Confidence     float64        `json:"confidence"`     // 0.0–1.0 score for how strongly this category matched
	Rule           string         `json:"rule"`           // name of the rule that produced this label
	Evidence       []string       `json:"evidence"`       // matched keyword snippets supporting this label
}

// ClassifiedDocument pairs a normalized document with its detected classification and confidence.
type ClassifiedDocument struct {
	Document       NormalizedDocument `json:"document"`       // the normalized artifact being classified
	Classification Classification     `json:"classification"` // the best matching signal category
	Confidence     float64            `json:"confidence"`     // 0.0–1.0 score for how certain the classification is
	Labels         []ScoredLabel      `json:"labels"`         // every category the document matched, ordered by confidence
	MatchedRules   []string           `json:"matched_rules"`  // names of all classification rules that fired
	Evidence       []string           `json:"evidence"`       // keyword snippets supporting the primary classification
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
	ID               string            `json:"id"`                // unique identifier combining source and canonical key
	Type             EntityType        `json:"type"`              // what kind of concept this entity is
	Name             string            `json:"name"`              // the surface form of the entity as it appeared in the text
	RawMention       string            `json:"raw_mention"`       // original text exactly as it appeared before normalization
	SourceID         string            `json:"source_id"`         // ID of the document this entity was extracted from
	Confidence       float64           `json:"confidence"`        // 0.0–1.0 score for how certain the extraction is
	ExtractionMethod string            `json:"extraction_method"` // how this entity was extracted, e.g. "regex_token" or "openapi"
	Spans            []SourceSpan      `json:"spans"`             // offsets or structured pointers locating the mention in the source
	Aliases          []string          `json:"aliases"`           // all known name variants merged during identity resolution
	Metadata         map[string]string `json:"metadata"`          // additional context attached during extraction
}

// RelationshipKind is the stable vocabulary describing how two entities relate.
type RelationshipKind string

const (
	// CoOccursInDocument links two entities that appeared together in the same source artifact.
	CoOccursInDocument RelationshipKind = "co_occurs_in_document"
	// RequirementAffectsAPI links a requirement to an API field it constrains or drives.
	RequirementAffectsAPI RelationshipKind = "requirement_affects_api"
	// RequirementAffectsService links a requirement to a service responsible for delivering it.
	RequirementAffectsService RelationshipKind = "requirement_affects_service"
	// APIBackedByDB links an API field to the database column that persists it.
	APIBackedByDB RelationshipKind = "api_backed_by_db"
	// EnumConstrainsField links an enum to the API field or column whose values it restricts.
	EnumConstrainsField RelationshipKind = "enum_constrains_field"
	// ServiceDependsOn links a service to a dependency it relies on.
	ServiceDependsOn RelationshipKind = "service_depends_on"
)

// Relationship describes a directed link between two entities.
type Relationship struct {
	ID         string            `json:"id"`         // deterministic ID built from the two entity IDs and kind
	FromID     string            `json:"from_id"`    // ID of the source entity
	ToID       string            `json:"to_id"`      // ID of the target entity
	Kind       RelationshipKind  `json:"kind"`       // typed label describing how the two entities relate
	Confidence float64           `json:"confidence"` // 0.0–1.0 score for how certain the edge is
	Evidence   []string          `json:"evidence"`   // source artifact references supporting this edge
	Metadata   map[string]string `json:"metadata"`   // extra context attached by the relationship builder
}

// Mismatch describes a detected delivery misalignment between teams or artifacts.
type Mismatch struct {
	ID            string   `json:"id"`             // unique identifier for this mismatch finding
	Type          string   `json:"type"`           // stable category for the detection rule that produced this finding
	Summary       string   `json:"summary"`        // human-readable description of what was found
	EntityIDs     []string `json:"entity_ids"`     // IDs of the entities involved in the mismatch
	Severity      string   `json:"severity"`       // impact level: low, medium, or high
	Confidence    float64  `json:"confidence"`     // 0.0-1.0 score for how certain the reasoning stage is
	Impact        string   `json:"impact"`         // expected business or delivery impact level
	Evidence      []string `json:"evidence"`       // source artifact references supporting this finding
	AffectedRoles []string `json:"affected_roles"` // delivery roles that need to act on this finding
	Recommended   string   `json:"recommended"`    // suggested action for the team to resolve this
}
