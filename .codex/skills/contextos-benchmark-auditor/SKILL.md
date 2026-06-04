---
name: contextos-benchmark-auditor
description: "Audit the ContextOS misalignment benchmark for product correctness. Use when: auditing benchmark quality; proving ContextOS finds real cross-layer contradictions; adding misalignment benchmark cases; evaluating precision, recall, evidence accuracy, false positives, severity calibration, or deterministic stability. Covers benchmark health, contradiction-focused cases, and fixture-backed scoring."
argument-hint: "Which misalignment benchmark suite, case, or product claim should be audited?"
user-invocable: true
---

# ContextOS Benchmark Auditor

## Outcome

Deliver a correctness audit for the core ContextOS misalignment benchmark with:

- `tests/harness/scenarios/reasoning/` — scenario YAMLs that prove contradiction detection, omissions, stale docs, negative controls, ambiguity handling, and evidence accuracy.
- `tests/harness/fixtures/reasoning/<case-id>/input.txt` and `source-metadata.json` — local deterministic text fixtures for every benchmark case.
- `tests/harness/golden/reasoning/<case-id>.json` — semantic golden expectations with evidence, confidence, severity, and metric thresholds.
- `.codex/skills/contextos-benchmark-auditor/assets/misalignment-benchmark-case-skeleton.md` — copyable case skeleton for new benchmark entries.
- `.codex/skills/contextos-benchmark-auditor/references/benchmark-audit-checklist.md` — audit checklist and v1 case catalog.

## When to Use

| Situation | Apply this skill with |
| --- | --- |
| Auditing whether ContextOS is really finding cross-layer contradictions | `contextos-misalignment-report` for finding semantics |
| Adding or reviewing text-fixture misalignment scenarios | `contextos-harness-engineering` for scenario, fixture, and golden layout |
| Evaluating precision, recall, false positives, or stability | Existing Go harness commands and the audit checklist |
| Reviewing an ambiguous or negative-control case | Needs-review rules from `contextos-misalignment-report` |

## Benchmark Health Dimensions

Score benchmark health across these dimensions before calling the benchmark credible:

- Precision: emitted mismatches are real contradictions, omissions, stale-doc findings, or calibrated needs-review items.
- Recall: expected contradictions from the catalog are found.
- Evidence accuracy: evidence points to the artifact and claim that actually proves the mismatch, not a nearby keyword.
- False positives: clean, ambiguous, and false-friend cases do not become hard mismatches.
- Severity calibration: high severity is reserved for strong evidence with material delivery impact.
- Deterministic stability: local fixtures, sorted semantic goldens, stable thresholds, and repeatable Go harness results.

## Procedure

1. **Frame the claim** — state the product claim being audited: ContextOS detects real cross-layer context contradictions with evidence, not generic summaries.
2. **Select case mix** — use the v1 catalog in [benchmark-audit-checklist.md](./references/benchmark-audit-checklist.md), keeping positive, negative, ambiguous, false-positive guard, severity, and evidence-accuracy cases balanced.
3. **Create cases** — start from [misalignment-benchmark-case-skeleton.md](./assets/misalignment-benchmark-case-skeleton.md) and place executable cases under `tests/harness/scenarios/reasoning/`, `tests/harness/fixtures/reasoning/`, and `tests/harness/golden/reasoning/`.
4. **Apply harness rules** — apply the **contextos-harness-engineering** skill for fixture paths, scenario contracts, golden output rules, and deterministic run commands.
5. **Apply finding rules** — apply the **contextos-misalignment-report** skill for contradiction types, evidence, confidence, impact, severity, and needs-review calibration.
6. **Score health** — record precision, recall, false-positive rate, evidence accuracy, severity calibration, and deterministic stability using the audit checklist.
7. **Document alignment** — update `tests/harness/README.md` and the nearest README when case layout, run commands, benchmark meaning, or routing changes.
8. **Validate** — run `GOCACHE=/tmp/context-os-gocache go test ./tests`, `GOCACHE=/tmp/context-os-gocache go test ./...`, `GOCACHE=/tmp/context-os-gocache go vet ./...`, and the authoring checks when the skill or routing changes.

## Decision Points

| Finding | Classification |
| --- | --- |
| Strong contradiction across requirement, code, API, docs, ticket, or frontend evidence | Hard mismatch with evidence, confidence, impact, and severity |
| Required artifact or field is absent where a source explicitly requires it | Omission mismatch |
| Documentation describes a route, request, or behavior contradicted by implementation | Stale-doc mismatch |
| Text is vague, lacks a measurable target, or supports multiple interpretations | No hard mismatch; optional needs-review only if the current output supports that class |
| Keyword overlap without a contradictory claim | No mismatch |
| Evidence cites an artifact that mentions the concept but does not prove the contradiction | Evidence failure, even if the mismatch type is correct |

## Do Not

- Do not treat generic summaries, topical overlap, keyword hits, or low-confidence suspicion as benchmark success.
- Do not add a benchmark case unless it tests contradiction detection, omission detection, stale documentation, ambiguity restraint, false-positive restraint, severity calibration, or evidence accuracy.
- Do not create live-service, nondeterministic, or time-sensitive benchmark fixtures.
- Do not hide a regression by loosening thresholds or regenerating goldens without explaining the product behavior change.

## References

- [Misalignment Benchmark Case Skeleton](./assets/misalignment-benchmark-case-skeleton.md)
- [Benchmark Audit Checklist](./references/benchmark-audit-checklist.md)
