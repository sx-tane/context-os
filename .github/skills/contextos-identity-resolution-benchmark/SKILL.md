---
name: contextos-identity-resolution-benchmark
description: "Design or run identity-resolution evaluation for aliases, semantic matches, multilingual names, and merge conflicts in ContextOS. Use when adding merge rules, tuning thresholds, reviewing precision and recall, evaluating multilingual entity matching, or stress-testing canonical identity linking."
argument-hint: "Which entity family or dataset should be benchmarked?"
user-invocable: true
---
# ContextOS Identity Resolution Benchmark

## Outcome
A repeatable, evidence-backed evaluation of canonical entity linking quality across all five matching layers.

## When to Use
- Adding or changing identity merge rules.
- Evaluating precision and recall before or after layer changes.
- Reviewing multilingual or naming-convention drift.
- Stress-testing the identity module after a connector change.
- Validating human-confirmation thresholds before a release.

## Identity Matching Layers

All five layers run in order. A later layer only activates if the earlier layer could not produce a confident match.

| Layer | Strategy | Confidence basis |
|-------|----------|-----------------|
| 1 — Exact | String equality after normalization | Deterministic |
| 2 — Semantic | Embedding cosine similarity | Score vs threshold |
| 3 — Relationship | Shared API, service, or screen co-occurrence | Graph proximity |
| 4 — Historical memory | Prior merge or rename events | Provenance |
| 5 — Human confirmation | Mandatory review for high-impact ambiguous merges | Manual decision |

## Procedure

### 1. Scope the benchmark
- Choose a target entity family (e.g. refund fields, order statuses, user roles).
- List all known aliases from connected sources (code, tickets, Slack, OpenAPI, spreadsheets).
- Define the expected canonical identity for each alias.

### 2. Build the dataset
- Use [benchmark-dataset-template.csv](./assets/benchmark-dataset-template.csv) to structure pairs.
- Include at minimum: exact positives, semantic positives, multilingual positives, and negative pairs.
- Tag each pair with the expected matching layer.

### 3. Run the benchmark
- Execute: `go test ./internal/identity/... -run BenchmarkIdentityResolution -v`
- See [benchmark-runner.go](./scripts/benchmark-runner.go) for the evaluation harness.

### 4. Record metrics
- Capture per-layer and overall results using the [Evaluation Matrix](./references/evaluation-matrix.md).
- Capture false merges, conflicts, and unresolved cases separately.

### 5. Analyze conflicts
- For each conflict, use the [Conflict Decision Tree](./references/conflict-decision-tree.md) to classify and route.

### 6. Propose updates
- Write specific rule additions, threshold changes, or block rules with evidence.
- Tag high-risk changes (may increase false merge rate) for human review.

## Decision Points
- Confidence low + impact high → route to Layer 5 human confirmation.
- Semantic match contradicts relationship evidence → keep identities separate and flag.
- Repeated false merge pattern detected → add explicit block rule before tuning thresholds.
- Layer 4 historical merge conflicts with current evidence → prefer current evidence and log divergence.

## Completion Checks
- Benchmark set covers all five layers with positive and negative cases.
- Precision, recall, and conflict rate are measured per layer.
- Every conflict has a classification and routing note.
- Proposed threshold changes carry an estimated recall risk.
- Results are reproducible from the same snapshot and dataset file.

## References
- [Evaluation Matrix](./references/evaluation-matrix.md)
- [Conflict Decision Tree](./references/conflict-decision-tree.md)
- [Benchmark Dataset Template](./assets/benchmark-dataset-template.csv)
- [Benchmark Runner Script](./scripts/benchmark-runner.go)
