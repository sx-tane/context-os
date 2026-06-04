# Benchmark Audit Checklist

Run before calling the misalignment benchmark credible.

## Dataset Balance

1. [ ] Includes clean agreement negative controls where all layers use the same route, field, or behavior.
2. [ ] Includes omission cases where a required field, source URI, evidence link, or behavior is absent.
3. [ ] Includes contract drift cases spanning at least two layers such as ticket, API, frontend, README, or handler.
4. [ ] Includes stale documentation cases where docs contradict implementation.
5. [ ] Includes ambiguity cases where vague requirements must not become hard mismatches.
6. [ ] Includes false-friend keyword cases that protect against topical or lexical false positives.
7. [ ] Includes severity calibration cases with high-impact evidence loss or delivery risk.
8. [ ] Includes evidence accuracy guard cases where nearby mentions are insufficient proof.

## Evidence Quality

1. [ ] Every expected mismatch has evidence that proves the contradiction, omission, or stale-doc claim.
2. [ ] Evidence points to the responsible artifact or behavior, not merely to a file that mentions a keyword.
3. [ ] Golden outputs assert `evidence`, `confidence_min`, `severity`, `impact`, and `entity_names` for every mismatch.
4. [ ] Negative controls and ambiguity cases expect no hard mismatch.
5. [ ] Needs-review expectations are used only when the current reasoning contract supports them.

## Metric Gates

1. [ ] Every executable scenario sets `precision_min`, `recall_min`, and `false_positive_rate_max`.
2. [ ] Positive mismatch cases use `precision_min: 1.00` and `recall_min: 1.00` unless a documented partial-recall baseline exists.
3. [ ] Negative, ambiguous, and false-friend cases use `false_positive_rate_max: 0.00`.
4. [ ] Whole-benchmark review reports precision, recall, evidence accuracy, false-positive rate, severity calibration, and deterministic stability.
5. [ ] Threshold changes are justified by an intentional product behavior change, not by harness instability.

## Regression Stability

1. [ ] Fixtures are local text files with no live services, network calls, clocks, or environment-specific paths.
2. [ ] Scenario IDs are stable kebab-case names and match fixture/golden filenames.
3. [ ] Golden files are semantic expectations with deterministic ordering and no timestamps or generated IDs.
4. [ ] The benchmark can run with `GOCACHE=/tmp/context-os-gocache go test ./tests`.
5. [ ] The full Go suite and vet command are run after executable scenarios or reasoning behavior changes.

## V1 Case Catalog

Use these text-fixture cases as the initial catalog. Implement them under `tests/harness/scenarios/reasoning/`, `tests/harness/fixtures/reasoning/`, and `tests/harness/golden/reasoning/` when the current harness behavior can satisfy the expected semantics.

| Case ID | Purpose | Example input text | Expected result |
| --- | --- | --- | --- |
| `spec-code-clean-agreement` | Negative control | `Spec says checkoutStatus is returned by API. Backend returns checkoutStatus. Frontend displays checkoutStatus.` | No mismatch |
| `spec-code-missing-field` | Omission detection | `Spec requires source_uri in every chat answer section. Handler response includes summary and evidence but drops source_uri.` | One mismatch with evidence pointing to `source_uri` omission |
| `ticket-api-route-drift` | Contract drift | `Ticket says POST /presentation/findings. Frontend calls POST /findings. API exposes POST /presentation/findings.` | One mismatch: frontend route contradicts API/ticket |
| `readme-api-stale-doc` | Stale documentation | `README says run /chat/query for answers. API route is /chat and request type is ChatQuery.` | One stale-doc mismatch |
| `ambiguous-performance-requirement` | Avoid overclaiming | `PM says analysis should be fast enough. No latency target is defined. Code runs analysis synchronously.` | No hard mismatch; optional needs-review only if supported |
| `false-friend-keyword-clean` | False-positive guard | `The term missing appears in MissingLink component name, but the feature is implemented and documented consistently.` | No mismatch |
| `severity-high-data-loss` | Severity calibration | `Requirement says preserve source evidence links. Normalization removes all evidence references before reasoning.` | High severity mismatch with evidence |
| `evidence-wrong-line-guard` | Evidence accuracy | `Spec and code disagree about cancellation. Only the handler file mentions ctx.Done; stage code ignores it.` | Mismatch evidence must point to the stage behavior, not only the handler mention |

## Final

1. [ ] `tests/harness/README.md` documents the expected case mix, score dimensions, layout, and run command.
2. [ ] `.codex/README.md` lists `contextos-benchmark-auditor` when skill routing changes.
3. [ ] `.codex/agents/contextos-implementer.agent.md` wires this skill together with `contextos-harness-engineering`.
4. [ ] Authoring validation passes after skill or routing changes:
   - `.codex/skills/contextos-authoring/scripts/score-skills.sh`
   - `.codex/skills/contextos-authoring/scripts/score-skill-routing.sh`
   - `.codex/skills/contextos-authoring/scripts/check-mermaid-policy.sh`
   - `.codex/skills/contextos-authoring/scripts/score-readme-coverage.sh`
   - `.codex/skills/contextos-authoring/scripts/score-readme-quality.sh`
   - `.codex/skills/contextos-authoring/scripts/check-readme-sync-on-change.sh`
5. [ ] Harness validation passes after executable benchmark scenarios are added:
   - `GOCACHE=/tmp/context-os-gocache go test ./tests`
   - `GOCACHE=/tmp/context-os-gocache go test ./...`
   - `GOCACHE=/tmp/context-os-gocache go vet ./...`
   - `git diff --check`
