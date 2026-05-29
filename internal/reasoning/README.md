# Reasoning Domain

The reasoning domain analyzes the context graph and emits delivery mismatch findings.

## Responsibility

- Inspect canonical entities and relationships.
- Detect cross-layer context misalignment and delivery understanding gaps.
- Produce actionable `types.Mismatch` findings.

## Input And Output

```mermaid
flowchart LR
  ctxgraph[ContextGraph]
  detect[DetectMismatches]
  mismatches["[]Mismatch"]

  ctxgraph --> detect --> mismatches
```

## Key API

```go
func DetectMismatches(g EntityReader) []types.Mismatch
```

`EntityReader` is the narrow interface consumed by reasoning:

```go
type EntityReader interface {
  AllEntities() []entities.CanonicalEntity
}
```

## Current Detection Rule

For every canonical entity in the graph, lowercase the entity name and emit a mismatch when it contains any of:

- `missing`
- `mismatch`
- `outdated`

The emitted finding uses:

- `ID`: `mismatch:<entity id>`
- `Type`: `keyword_signal`
- `Summary`: `Potential delivery mismatch around <entity name>`
- `EntityIDs`: the single matching entity ID
- `Severity`: `medium`
- `Confidence`: `0.70`
- `Impact`: `medium`
- `Evidence`: source URI plus entity fragment when available, otherwise the entity source ID
- `Recommended`: confirmation guidance across knowledge layers (presentation, service, PMO)

## Dependencies

```mermaid
flowchart TD
  reasoning[internal/reasoning]
  ctxgraph[internal/graph]
  entities[domain/entities]
  types[domain/types]
  presentation[internal/presentation]
  pipeline[domain/pipelines]

  reasoning -. EntityReader .-> ctxgraph
  reasoning --> entities
  reasoning --> types
  presentation --> reasoning
  pipeline --> reasoning
```

## Example Usage

```go
mismatches := reasoning.DetectMismatches(contextGraph)
```

## Implementation Notes

- This is intentionally explainable but not production-complete. It is the first deterministic rule for cross-layer context drift detection.
- Findings include confidence, impact, and evidence back to source artifacts.
- Avoid opaque AI-only findings. AI output should support or rank evidence, not replace provenance.
- Add tests for every detection rule because reasoning changes directly affect the first production success metric.

## Production Requirements

- Detect cross-layer contract drift, PMO status drift, requirement gaps, outdated implementation assumptions, and dependency risks.
- Emit confidence, impact, affected roles, evidence, severity, and recommended action for every finding.
- Keep false-positive tracking and regression fixtures for real or realistic delivery artifacts.
- Treat execution output as supporting evidence only when it can be traced and reviewed.
