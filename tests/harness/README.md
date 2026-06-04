# ContextOS Harnesses

This directory is the shared home for deterministic pipeline, benchmark, and regression scenarios.

## Layout

```text
tests/harness/
  scenarios/<area>/<scenario-name>.yaml
  fixtures/<area>/<scenario-name>/
  golden/<area>/<scenario-name>.json
```

Use package-local `internal/<stage>/testdata` when a harness only proves one stage. Use this shared directory when a scenario crosses stage boundaries or should become a long-lived regression case.

## Scenario Rules

- Start from `.github/skills/contextos-harness-engineering/assets/scenario-template.yaml`.
- Keep inputs local and deterministic.
- Sort outputs before comparing with golden files.
- Prefer semantic assertions when exact snapshots are brittle.
- Document any intentional golden or metric baseline update in the change summary.

## Run Commands

Shared pipeline harness coverage lives in `tests/pipeline_test.go` and loads scenarios from `tests/harness/scenarios/`:

```sh
go test ./tests
```

Relationship benchmark scenarios live under `tests/harness/scenarios/relationship/` with fixtures
and goldens in matching `relationship/` folders. They run both deterministic baseline mode and a
fake assistant mode; real Codex CLI calls are not part of harness tests. Relationship metrics score
semantic relationship kinds and ignore raw `co_occurs_in_document` edges.

Relationship benchmark gates use:

- `relationship_precision_min`
- `relationship_recall_min`
- `relationship_false_positive_rate_max`

## Misalignment Benchmark Audit

Misalignment benchmark quality is audited with `.codex/skills/contextos-benchmark-auditor/` and implemented through the existing reasoning harness layout:

```text
tests/harness/
  scenarios/reasoning/<case-id>.yaml
  fixtures/reasoning/<case-id>/
    input.txt
    source-metadata.json
  golden/reasoning/<case-id>.json
```

The expected case mix covers clean agreement, missing required fields, ticket/API/frontend route drift, stale README/API claims, ambiguous performance requirements, false-friend keyword guards, high-severity evidence loss, and evidence-location guards. Every executable case should test contradiction detection or calibrated restraint, not generic summarization.

Audit score dimensions are:

- `precision`
- `recall`
- `false_positive_rate`
- evidence accuracy
- severity calibration
- deterministic stability

Run reasoning misalignment scenarios through the shared harness:

```sh
GOCACHE=/tmp/context-os-gocache go test ./tests
```

Run the full Go suite after changing scenarios, fixtures, goldens, or stage behavior:

```sh
go test ./...
```

## Golden Update Policy

Golden files under `tests/harness/golden/` are semantic expectations, not raw full snapshots. Update them only when the source behavior intentionally changes, keep arrays sorted, omit volatile values such as timestamps and generated IDs, and describe the reason for the golden change in the implementation summary.
