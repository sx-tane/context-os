# Frontend Routes

This folder contains Svelte route files and page entrypoints.

## Files

- `+page.svelte`: main local ContextOS UI page that probes service status, renders connector cards, and coordinates ingest actions through `src/lib` helpers.

## Responsibilities

- Define page-level data flow and actions.
- Compose connector components and shared UI blocks.
- Keep route state transitions understandable and testable.

## Maintenance Checklist

- Document significant route behavior changes here.
- Keep route integration tests aligned with UI behavior.
- Update linked component docs for new props or events.
