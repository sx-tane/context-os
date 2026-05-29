---
name: contextos-harness-engineering
description: "Design, implement, or review deterministic ContextOS test/evaluation harnesses. Use when: creating fixtures or testdata; adding scenarios or golden outputs; designing benchmark baselines or regression gates. Covers harness levels, fixture layout, scenario contracts, metrics, and golden update policy."
argument-hint: "Which stage, connector, or end-to-end behavior should the harness cover?"
user-invocable: true
---

# ContextOS Harness Engineering

## Outcome

A deterministic harness that turns ContextOS behavior into repeatable scenarios with explicit inputs, expected outputs, metrics, and regression gates.

## When to Use

- Adding or refactoring pipeline, connector, reasoning, identity, or cross-layer behavior that needs more than a single unit test.
- Creating fixtures, `testdata`, golden outputs, benchmarks, or baseline comparisons.
- Turning a bug, mismatch report, connector sample, or product example into a reusable regression scenario.
- Reviewing whether test coverage proves behavior across stage boundaries.
- Designing CI gates for precision, recall, conflict rate, or output stability.

## Harness Levels

Choose the narrowest harness level that proves the behavior.

| Level      | Use for                                                  | Required proof                                          |
| ---------- | -------------------------------------------------------- | ------------------------------------------------------- |
| Unit       | Pure functions, parsing helpers, small rules             | Direct expected values and error paths                  |
| Stage      | One internal stage or connector                          | Input fixture, normalized output, traceability fields   |
| Pipeline   | Multi-stage behavior through `internal/pipeline`         | Scenario file, expected entities/relationships/findings |
| Benchmark  | Quality tuning, identity resolution, reasoning precision | Dataset, metrics, baseline, threshold                   |
| Regression | Past bug or customer-like case                           | Frozen input, golden output, owner note                 |

## Fixture Layout

Use this layout unless an existing package already has a stronger local convention:

```text
tests/
  harness/
    README.md
    scenarios/
      <area>/<scenario-name>.yaml
    fixtures/
      <area>/<scenario-name>/
        input.*
        source-metadata.json
    golden/
      <area>/<scenario-name>.json
```

For package-local harnesses, use:

```text
internal/<stage>/testdata/
  scenarios/
  fixtures/
  golden/
```

## Scenario Contract

Every scenario file must include:

- `id`: stable kebab-case identifier.
- `area`: one of the ContextOS area labels (`source`, `ingestion`, `normalization`, `classification`, `extraction`, `identity`, `relationship`, `graph`, `reasoning`, `execution`, `presentation`, `contracts`).
- `level`: `unit`, `stage`, `pipeline`, `benchmark`, or `regression`.
- `description`: one sentence describing the behavior.
- `inputs`: fixture paths or inline source requests.
- `expected`: entity, relationship, event, mismatch, status, or metric expectations.
- `evidence`: artifact references that justify the expectation.
- `thresholds`: metric gates when exact equality is too brittle.
- `owner`: owning stage or feature.

Start new scenarios from [scenario-template.yaml](./assets/scenario-template.yaml).

## Golden Output Rules

- Golden files must be deterministic: sort arrays, omit timestamps, and normalize generated IDs.
- Prefer semantic assertions over full-file snapshots when only part of the output matters.
- Include `Evidence []string` and `Confidence float64` expectations for reasoning outputs.
- Golden updates must be intentional and described in the change summary.
- Do not hide behavior changes by regenerating goldens without explaining the source change.

## Metrics

Use exact assertions for deterministic transformations. Use metrics for quality-sensitive behavior:

- `precision`: accepted outputs that are correct.
- `recall`: expected outputs that were found.
- `false_positive_rate`: outputs produced but not expected.
- `unresolved_rate`: expected items that the pipeline could not resolve.
- `conflict_rate`: contradictory or ambiguous outputs.

Benchmark and quality harnesses must record previous baseline, new result, and pass/fail threshold.

## Procedure

1. Define the behavior and choose the harness level.
2. Add the smallest fixture set that proves normal, error, and ambiguity or conflict paths.
3. Create or update scenario files using the scenario contract.
4. Implement the harness runner or test wrapper with deterministic ordering.
5. Assert traceability: source URI, event IDs, evidence references, and confidence where applicable.
6. Add golden outputs or metric thresholds only where they reduce ambiguity.
7. Document the run command, golden update policy, and fixture/scenario layout in the nearest README.
8. Run the target harness plus relevant `go test` packages.

## Decision Points

- If a full pipeline harness fails but stage harnesses pass: add an integration scenario at the boundary where assumptions diverge.
- If output order is unstable: sort by stable identifiers before asserting.
- If output content is nondeterministic: expose a deterministic dependency or assert semantic fields only.
- If a scenario depends on live external services: replace it with recorded fixtures or a fake connector.
- If exact golden equality makes valid improvements painful: switch to structured assertions plus metric thresholds.

## Completion Checks

- Scenario can be rerun from local files without network access.
- Fixture inputs and expected outputs are committed together.
- Tests cover at least one negative, ambiguity, or regression path for non-trivial behavior.
- Metrics have named thresholds and baselines when used.
- Run command is documented in the relevant README.
- Fixture, scenario, and golden locations are documented in the relevant README when new folders are introduced.
- Results are reproducible from the same repo snapshot.

## References

- [Scenario Template](./assets/scenario-template.yaml)
- [Harness Checklist](./references/harness-checklist.md)
