# Domain Entities

Package `domain/entities` contains the canonical entity wrapper produced by identity resolution.

## Responsibility

Represent the result of resolving raw extracted entities into canonical identities. This package intentionally wraps `domain/types.Entity` instead of redefining the entity itself.

## Key Type

```go
type CanonicalEntity struct {
    Entity     types.Entity `json:"entity"`
    Confidence float64      `json:"confidence"`
    NeedsHuman bool         `json:"needs_human"`
}
```

## Field Meaning

| Field | Meaning |
| --- | --- |
| `Entity` | Canonical entity payload, including aliases and metadata. |
| `Confidence` | Resolution confidence from 0 to 1. Current exact canonical-key grouping uses `1`. |
| `NeedsHuman` | Manual-review flag for ambiguous merges. Current exact matching sets this to `false`. |

## Produced By

[internal/identity](../../internal/identity/README.md) produces canonical entities from extracted `types.Entity` values.

```mermaid
flowchart LR
  extracted[[]types.Entity]
  identity[identity.Resolve]
  canonical[[]entities.CanonicalEntity]

  extracted --> identity --> canonical
```

## Implementation Notes

- Keep confidence explainable. Future semantic or multilingual matching should include evidence in metadata or a richer contract before lowering certainty.
- `NeedsHuman` is the hook for conflict review once identity resolution moves beyond deterministic key matching.
- Canonical entities are stored in [internal/graph](../../internal/graph/README.md).
