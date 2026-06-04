# Misalignment Benchmark Case Skeleton

Copy this skeleton when adding a new misalignment benchmark case under the shared harness. Delete every `<!-- only if ... -->` annotation after applying it.

## Directory Layout

```text
tests/harness/
  scenarios/reasoning/<case-id>.yaml
  fixtures/reasoning/<case-id>/
    input.txt
    source-metadata.json
  golden/reasoning/<case-id>.json
```

## Fixture: `input.txt`

```text
<source text that contains the smallest cross-layer claim set needed to prove the case>

<!-- only if the case is a positive mismatch -->
Include at least two artifacts or role views that contradict each other, such as spec versus handler, ticket versus frontend route, or README versus API route.

<!-- only if the case is a negative control -->
Include consistent claims across the compared layers and avoid unsupported bait keywords.

<!-- only if the case is ambiguous -->
Use vague wording with no measurable target, then expect no hard mismatch.
```

## Fixture: `source-metadata.json`

```json
{
  "uri": "repo://benchmark/<case-id>",
  "source": "text-fixture",
  "area": "reasoning",
  "artifact_kind": "misalignment-benchmark",
  "case_id": "<case-id>"
}
```

## Scenario: `<case-id>.yaml`

```yaml
id: <case-id>
area: reasoning
level: benchmark
description: "<one sentence naming the contradiction or guard this case proves>"
owner: internal/reasoning

inputs:
  source_request:
    uri: "repo://benchmark/<case-id>"
    content_path: "tests/harness/fixtures/reasoning/<case-id>/input.txt"
    metadata_path: "tests/harness/fixtures/reasoning/<case-id>/source-metadata.json"

expected:
  golden_path: "tests/harness/golden/reasoning/<case-id>.json"
  mismatches:
    # <!-- only if expected mismatch count is nonzero -->
    - summary: "<semantic mismatch summary>"
      evidence:
        - "repo://benchmark/<case-id>#<claim-or-token>"
      confidence_min: 0.70

thresholds:
  precision_min: 1.00
  recall_min: 1.00
  false_positive_rate_max: 0.00

evidence:
  - "tests/harness/fixtures/reasoning/<case-id>/input.txt"
  - "tests/harness/fixtures/reasoning/<case-id>/source-metadata.json"

notes: "<why this case belongs in the benchmark and what false-positive or false-negative risk it covers>"
```

## Golden: `<case-id>.json`

```json
{
  "entities": [
    {
      "name": "<stable claim, field, route, or artifact token>",
      "type": "<semantic entity type>"
    }
  ],
  "mismatches": [
    {
      "type": "<contradiction|omission|stale_doc|needs_review>",
      "summary": "<short semantic summary>",
      "severity": "<low|medium|high|critical|needs-review>",
      "confidence_min": 0.7,
      "impact": "<low|medium|high>",
      "evidence": [
        "repo://benchmark/<case-id>#<claim-or-token>"
      ],
      "entity_names": [
        "<entity name tied to the mismatch>"
      ]
    }
  ]
}
```

For clean negative controls, use an empty `mismatches` array and keep `false_positive_rate_max: 0.00`.
