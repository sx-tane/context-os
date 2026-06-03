# Harness Checklist

Use this before marking harness work complete.

## Scope

- [ ] Harness level is explicit: unit, stage, pipeline, benchmark, or regression.
- [ ] Scenario area maps to a ContextOS domain or source connector.
- [ ] The scenario proves a user-visible or downstream-relevant behavior.

## Fixtures

- [ ] Inputs are local fixtures or deterministic fakes.
- [ ] No live network, clock, random, or filesystem dependency leaks into assertions.
- [ ] Fixture names are stable and describe the behavior.

## Assertions

- [ ] Output ordering is deterministic before comparison.
- [ ] Golden files omit volatile fields or normalize them before comparison.
- [ ] Reasoning outputs assert `Evidence []string` and `Confidence float64` when present.
- [ ] Negative, ambiguity, or regression path is covered for non-trivial behavior.

## Metrics

- [ ] Metric thresholds are named and justified.
- [ ] Baseline result and new result are recorded for benchmark changes.
- [ ] False positives and unresolved cases are tracked separately when relevant.

## Documentation

- [ ] Run command is documented in the package or harness README.
- [ ] Golden update policy is documented.
- [ ] Fixture, scenario, and golden locations are documented when new folders are introduced.
- [ ] Change summary explains any expected baseline or golden changes.
