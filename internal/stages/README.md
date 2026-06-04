# Pipeline Stages

The `internal/stages` tree contains the independent implementation packages for the ContextOS source-to-finding pipeline. Each stage should consume and emit stable [`domain`](../../domain/README.md) contracts, preserve traceability, and avoid importing sibling stage implementations.

## Stage Packages

| Package | Responsibility |
| --- | --- |
| [`ingestion/`](ingestion/README.md) | Converts source connector events into accepted pipeline inputs. |
| [`normalization/`](normalization/README.md) | Produces deterministic normalized documents and parsed side outputs. |
| [`classification/`](classification/README.md) | Assigns document categories with evidence and confidence. |
| [`extraction/`](extraction/README.md) | Extracts entities and source facts from normalized content. |
| [`identity/`](identity/README.md) | Resolves aliases and semantic matches into canonical identities. |
| [`relationship/`](relationship/README.md) | Builds evidence-backed relationships between canonical entities. |
| [`graph/`](graph/README.md) | Materializes graph nodes, edges, reads, cleanup, and snapshots. |
| [`reasoning/`](reasoning/README.md) | Detects cross-layer mismatch findings with evidence and confidence. |
| [`execution/`](execution/README.md) | Applies output and execution rules after reasoning. |
| [`presentation/`](presentation/README.md) | Shapes findings and summaries for API and UI consumers. |

## Stage Flow

```mermaid
flowchart LR
  ingestion --> normalization
  normalization --> classification
  classification --> extraction
  extraction --> identity
  identity --> relationship
  relationship --> graph
  graph --> reasoning
  reasoning --> execution
  execution --> presentation
```

[`../pipeline/`](../pipeline/README.md) owns sequencing and cross-stage wiring. Stage packages should stay deterministic and synchronous; callers decide concurrency and cancellation boundaries.

## Maintenance Checklist

- Update the matching stage README when behavior, contracts, events, or tests change.
- Keep stage packages independent; use `domain/` types or narrow local interfaces as bridges.
- Run the relevant stage tests after moving imports or changing stage behavior.
