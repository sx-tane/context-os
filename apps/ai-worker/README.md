# AI Worker App

Python worker surface for local-first AI-assisted classification, extraction, and reasoning tasks.

Production responsibility:

- run optional AI-assisted work without replacing deterministic evidence;
- preserve prompts, source references, outputs, and trace metadata;
- support replay and cancellation;
- return assistive evidence that reasoning can inspect and audit.

## Run Commands

```bash
cd apps/ai-worker
uv sync
uv run python health.py
```

## Maintenance Checklist

- Keep worker behavior assistive and explainable rather than authoritative.
- Document new worker entrypoints or runtime dependencies here.
- Preserve local-first execution assumptions when adding integrations.
