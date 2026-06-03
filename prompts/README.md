# Prompts

Prompt templates for local AI execution and assistive reasoning workflows.

`findings.md` is the active production prompt consumed by local findings and reasoning flows. Older cross-layer and entity-context draft prompts were removed so this folder has one clear source of truth for generated finding language.

## Files

| File | Responsibility | Update when |
| --- | --- | --- |
| [`findings.md`](findings.md) | Evidence-first finding language for local reasoning and presentation flows. | Finding format, evidence policy, confidence wording, or recommendation expectations change. |

## Responsibilities

- Store reusable prompt text for local AI-assisted features.
- Keep prompts aligned with evidence-first and explainable output rules.
- Avoid embedding secrets, environment-specific tokens, or hidden assumptions.

## Maintenance Checklist

- Update prompt docs when prompt roles or output expectations change.
- Keep prompt wording consistent with local-first product direction.
- Document any prompt that is consumed by automation or worker flows.
