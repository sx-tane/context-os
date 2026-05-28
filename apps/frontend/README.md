# Frontend App

SvelteKit application surface for ContextOS presentation views.

Current local workflow:

- probes API and worker health from the main page;
- shows Codex CLI install/login/plugin status;
- supports Codex device-auth login and plugin re-auth streams;
- runs GitHub MCP ingestion through either direct token/env auth or the Codex GitHub plugin;
- runs Slack MCP ingestion through direct token/env/OAuth auth or the Codex Slack plugin;
- renders ingestion events, previews, metadata, and SSE progress logs.

Production responsibility:

- show role-specific PMO, frontend, backend, QA, and architecture views;
- keep findings tied to evidence and impacted artifacts;
- separate facts, confidence, impact, and recommended next actions;
- support fast local workflows for inspecting delivery misalignment.
