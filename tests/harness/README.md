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

Run the full Go suite after changing scenarios, fixtures, goldens, or stage behavior:

```sh
go test ./...
```

## Golden Update Policy

Golden files under `tests/harness/golden/` are semantic expectations, not raw full snapshots. Update them only when the source behavior intentionally changes, keep arrays sorted, omit volatile values such as timestamps and generated IDs, and describe the reason for the golden change in the implementation summary.
