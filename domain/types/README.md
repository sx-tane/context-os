# Domain Types

Package `domain/types` contains the serializable data structures shared across pipeline stages.

## Normalized Documents

```go
type SourceSpan struct {
    Field string `json:"field"`
    Start int    `json:"start"`
    End   int    `json:"end"`
    Line  int    `json:"line"`
    Path  string `json:"path"`
}

type NormalizedDocument struct {
    ID            string            `json:"id"`
    Source        string            `json:"source"`
    SourceType    string            `json:"source_type"`
    Title         string            `json:"title"`
    Body          string            `json:"body"`
    ContentHash   string            `json:"content_hash"`
    SchemaVersion string            `json:"schema_version"`
    Spans         []SourceSpan      `json:"spans"`
    Metadata      map[string]string `json:"metadata"`
    NormalizedAt  time.Time         `json:"normalized_at"`
}
```

Normalized documents are the common processing unit after ingestion. They should preserve source identity and enough metadata to trace back to the original artifact. `ContentHash` is the hex SHA-256 of the canonical title+body and enables replay and change detection; `SchemaVersion` is carried from the originating event; `Spans` locate canonical text back inside the source artifact via `SourceSpan`.

## Classification

```go
type Classification string

const (
    BusinessLogic   Classification = "business_logic"
    APIDiscussion   Classification = "api_discussion"
    PMORisk         Classification = "pmo_risk"
    ConsumerConcern Classification = "consumer_concern"
    ProducerConcern Classification = "producer_concern"
    Blocker         Classification = "blocker"
    Decision        Classification = "decision"
    Unknown         Classification = "unknown"
)
```

```go
type ScoredLabel struct {
    Classification Classification `json:"classification"`
    Confidence     float64        `json:"confidence"`
    Rule           string         `json:"rule"`
    Evidence       []string       `json:"evidence"`
}

type ClassifiedDocument struct {
    Document       NormalizedDocument `json:"document"`
    Classification Classification     `json:"classification"`
    Confidence     float64            `json:"confidence"`
    Labels         []ScoredLabel      `json:"labels"`
    MatchedRules   []string           `json:"matched_rules"`
    Evidence       []string           `json:"evidence"`
}
```

Classification routes a document toward domain-specific extraction and reasoning behavior. The current classifier is deterministic and confidence values are rule scores. `Labels` records every signal that fired (not just the winning one) so ambiguous multi-signal documents lose no information; `MatchedRules` and `Evidence` make each classification explainable.

## Entities

```go
type EntityType string

const (
    APIField    EntityType = "api_field"
    DBColumn    EntityType = "db_column"
    Enum        EntityType = "enum"
    Requirement EntityType = "requirement"
    Service     EntityType = "service"
    Dependency  EntityType = "dependency"
)
```

```go
type Entity struct {
    ID               string            `json:"id"`
    Type             EntityType        `json:"type"`
    Name             string            `json:"name"`
    RawMention       string            `json:"raw_mention"`
    SourceID         string            `json:"source_id"`
    Confidence       float64           `json:"confidence"`
    ExtractionMethod string            `json:"extraction_method"`
    Spans            []SourceSpan      `json:"spans"`
    Aliases          []string          `json:"aliases"`
    Metadata         map[string]string `json:"metadata"`
}
```

Entities are candidate or canonical domain concepts depending on stage. `SourceID` links back to the normalized document or source event. `RawMention` preserves the exact text fragment the entity came from, `Confidence` and `ExtractionMethod` explain how certain the extraction is and how it was produced, and `Spans` locate the mention inside the source artifact.

## Relationships

```go
type Relationship struct {
    ID       string            `json:"id"`
    FromID   string            `json:"from_id"`
    ToID     string            `json:"to_id"`
    Kind     string            `json:"kind"`
    Metadata map[string]string `json:"metadata"`
}
```

Relationships connect domain entities. The current implementation creates `co_occurs_in_document` relationships.

## Mismatches

```go
type Mismatch struct {
    ID          string   `json:"id"`
    Type        string   `json:"type"`
    Summary     string   `json:"summary"`
    EntityIDs   []string `json:"entity_ids"`
    Severity    string   `json:"severity"`
    Confidence  float64  `json:"confidence"`
    Impact      string   `json:"impact"`
    Evidence    []string `json:"evidence"`
    Recommended string   `json:"recommended"`
}
```

Mismatches are reasoning findings. Current findings include the detection type, confidence score, impact level, evidence references, severity, and recommended action so downstream presentation and regression harnesses can audit why a finding exists.

Production mismatch direction:

```go
type Mismatch struct {
    ID          string
    Summary     string
    EntityIDs   []string
    Severity    string
    Confidence  float64
    Impact      string
    Evidence    []string
    AffectedRoles []string
    Recommended string
}
```

Future expansions should add role-specific impact and recommendation status without removing the existing evidence and confidence fields.

## Pipeline Shape

```mermaid
flowchart LR
  doc[NormalizedDocument]
  classified[ClassifiedDocument]
  entity[Entity]
  canonical[CanonicalEntity]
  relationship[Relationship]
  mismatch[Mismatch]

  doc --> classified --> entity --> canonical --> relationship --> mismatch
```

## Maintenance Checklist

- Keep new shared types stable and JSON-friendly.
- Preserve `Evidence` and `Confidence` fields for reasoning outputs.
- Document breaking contract changes before updating downstream stages.
