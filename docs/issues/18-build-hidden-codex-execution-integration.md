# Issue 18: Build hidden Codex execution integration

## Description

Add internal execution flow that prepares context, runs Codex CLI, imports analysis, and updates the graph.

## Acceptance criteria

- Execution package defines a local Codex executor interface.
- Initial implementation is safe and local-first.
- Future implementation can replace the stub with real hidden Codex orchestration.
