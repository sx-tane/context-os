# Extraction Domain

The extraction domain turns classified document text into candidate entities.

## Responsibility

- Route a document to a structured extractor (OpenAPI, spreadsheet, Jira, GitHub, Codex labels) when its source metadata or content identifies one.
- Fall back to generic regex token extraction for unstructured or unparseable content, while skipping common prose tokens.
- Deduplicate candidates within one document.
- Infer a coarse entity type.
- Attach source spans, raw mention text, confidence, and extraction method to every entity.
- Preserve the source document ID and classification metadata.

## Input And Output

```mermaid
flowchart LR
  classified[ClassifiedDocument]
  extract[Extract]
  entities[[]Entity]

  classified --> extract --> entities
```

## Key API

```go
func Extract(doc types.ClassifiedDocument) []types.Entity
```

Internal helper:

```go
func inferType(name string, classification types.Classification) types.EntityType
```

## Routing

`Extract` inspects document metadata and dispatches to a structured extractor, falling back to
generic token extraction when no structured signal is present or a structured parse yields nothing:

| Metadata signal                    | Extractor         | Method label   |
| ---------------------------------- | ----------------- | -------------- |
| `filesystem_format = openapi_spec` | OpenAPI pointers  | `openapi`      |
| `filesystem_format = spreadsheet`  | Spreadsheet cells | `spreadsheet`  |
| `connector = jira` (JSON body)     | Jira `fields`     | `jira_field`   |
| `connector = github` (JSON body)   | GitHub top fields | `github_field` |
| `CONTEXTOS_LABELS_JSON:` line      | Codex labels      | `codex_label`  |
| otherwise                          | Regex tokens      | `regex_token`  |

Structured extractors return nothing for non-matching content, so the dispatcher safely falls back
to regex tokens (this keeps the reasoning harness deterministic for plain-text fixtures).

## Candidate Pattern

The token fallback keeps code and delivery identifiers such as camelCase, snake_case, issue keys,
and uppercase IDs. It deliberately skips common prose labels such as `and`, `also`, `Source`,
`Read`, `fields`, and generic lowercase `type` so old unstructured content does not dominate the
graph.

The token fallback uses this regular expression:

```text
[A-Z][A-Z0-9]+-\d+|[A-Za-z][A-Za-z0-9_]*(?:Status|State|ID|Id|Type|Flag|Field|Column)?|\b[A-Z]{2,}\d{2,}\b
```

Candidates shorter than three characters are ignored. Deduplication is case-insensitive within one document.

## Codex Labels

Codex-backed source reads are prompted to end with a single auditable label line:

```text
CONTEXTOS_LABELS_JSON: {"entities":{"requirement":[],"api_field":[],"service":[],"dependency":[],"enum":[],"db_column":[]},"risks":[],"decisions":[],"status":[]}
```

Each item carries `name`, `evidence`, and `confidence`. Parsed labels become `codex_label`
entities with `assistive=true`, the original source provenance, and evidence metadata. The labels
help reduce graph noise, but they remain assistive metadata rather than blind source of truth.

## Type Inference

| Condition                                                                 | Entity Type   |
| ------------------------------------------------------------------------- | ------------- |
| name contains `field`                                                     | `APIField`    |
| name contains `status` or `state` and looks schema-like (`refundStatus`) | `APIField`    |
| name contains `column` or `database`                                      | `DBColumn`    |
| name contains `type` or `flag`                                            | `Enum`        |
| document classification is `BusinessLogic`                                | `Requirement` |
| document classification is `APIDiscussion`                                | `Service`     |
| fallback                                                                  | `Dependency`  |

## Entity Shape

Each extracted entity receives:

- `ID`: document ID plus lowercase entity key.
- `Type`: inferred entity type.
- `Name`: normalized candidate text.
- `RawMention`: original source text before normalization.
- `SourceID`: normalized document ID.
- `Confidence`: extraction certainty (`0.58` for regex tokens, `0.9` for structured extractions unless a Codex label supplies confidence).
- `ExtractionMethod`: how the entity was produced (`regex_token`, `openapi`, `spreadsheet`, `jira_field`, `github_field`, `codex_label`).
- `Spans`: rune offsets (token path) or structured pointers/cell references (structured paths).
- `Metadata`: `classification`, plus `source_uri` and upstream `source_id` when present for downstream evidence.

## Dependencies

```mermaid
flowchart TD
  extraction[internal/extraction]
  types[domain/types]
  pipeline[domain/pipelines]

  extraction --> types
  pipeline --> extraction
```

## Example Usage

```go
extracted := extraction.Extract(classified)
```

## Implementation Notes

- The current extractor is intentionally deterministic and light. It is a candidate generator, not final truth.
- Keep `SourceID` intact because identity, relationship, and reasoning depend on provenance.
- Add tests before changing token behavior; extraction changes quickly affect graph shape and mismatch detection.

## Production Requirements

- Emit source spans or structured field paths for every extracted entity.
- Include extraction confidence and extraction method metadata.
- Support structured inputs such as Jira fields, filesystem documents, OpenAPI schemas, and spreadsheet cells.
- Preserve raw mention text separately from normalized entity names.

## Status

Source spans, raw mention text, confidence, and extraction-method metadata are implemented for the
token fallback, Codex labels, and the OpenAPI, spreadsheet, Jira, and GitHub structured extractors,
with tests in `extraction_test.go`.
