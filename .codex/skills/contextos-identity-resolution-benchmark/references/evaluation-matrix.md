# Identity Resolution Evaluation Matrix

## Per-Layer Metrics

| Layer | Precision target | Recall target | Acceptable conflict rate |
|-------|-----------------|--------------|--------------------------|
| 1 — Exact | 1.00 | — | 0 |
| 2 — Semantic | ≥ 0.90 | ≥ 0.85 | < 0.05 |
| 3 — Relationship | ≥ 0.85 | ≥ 0.80 | < 0.08 |
| 4 — Historical memory | ≥ 0.95 | — | < 0.02 |
| 5 — Human confirmation | 1.00 | 1.00 | 0 |

## Overall Metrics

- Overall precision: weighted average across resolved layers.
- Overall recall: aliases with a confirmed canonical match / total aliases.
- Conflict rate: cases where two layers disagree / total attempts.
- Human confirmation rate: cases escalated to Layer 5 / total attempts.
- False merge rate: incorrect merges confirmed after review / total merges.
- Unresolved rate: cases where no layer could produce a match / total attempts.

## Severity Tiers for False Merges

- Critical: distinct business entities merged (e.g. refund_status and payment_status).
- High: alias from a different service incorrectly linked.
- Medium: naming-convention variant from same domain incorrectly merged.
- Low: plural or case variant without semantic difference.

