---
description: "Use when implementing reasoning, graph analysis, or presentation outputs. Enforces explainable findings with confidence, impact, and evidence."
applyTo: "internal/{reasoning,presentation,graph}/**/*.go"
---
# Reasoning Output Instruction

- Findings should include mismatch type, confidence score, and impact level.
- Include evidence references to supporting source artifacts.
- Separate detection logic from rendering logic.
- Preserve deterministic ranking rules where possible.
- Track false-positive risk and unknown confidence states explicitly.
