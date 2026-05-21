---
name: contextos-misalignment-report
description: "Generate cross-layer context misalignment findings (presentation layer, service layer, PMO) with evidence, confidence, impact, and recommended actions."
argument-hint: "Which feature or artifact set should be analyzed?"
user-invocable: true
---

# ContextOS Misalignment Report

## Outcome

Create an explainable misalignment report linking assumptions to implementation evidence.

## When to Use

- Cross-layer contract drift checks (presentation layer vs service layer).
- PMO status vs implementation reality checks.
- Requirement gap analysis before release planning.

## Procedure

1. Select feature scope and gather source artifacts.
2. Extract claims or assumptions from each role view.
3. Compare claims against API, code, tickets, and discussions.
4. Identify contradiction, omission, and stale-assumption patterns.
5. Rank findings by impact and confidence.
6. Produce recommended remediation actions.

## Decision Points

- If evidence conflicts and confidence is low: classify as needs-review.
- If high-impact contradiction has strong evidence: classify as critical mismatch.
- If data is stale but not contradictory: classify as synchronization lag.

## Completion Checks

- Every finding includes evidence references.
- Confidence and impact are assigned consistently.
- Recommendations map to owner roles.
- Findings are reproducible from the same snapshot.

## References

- [Finding Severity Guide](./references/finding-severity-guide.md)
- [Report Template](./assets/misalignment-report-template.md)
