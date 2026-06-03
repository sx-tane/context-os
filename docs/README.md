# ContextOS Docs

Documentation for architecture, implementation issues, and local-first operating model.

## Start Here

- [Architecture](ARCHITECTURE.md) - production domain map, pipeline flow, data contracts, and links to every stage guide.
- [Production Readiness](PRODUCTION_READINESS.md) - issue-backed production acceptance criteria and stage-by-stage gaps.
- [MCP Connectors](mcp-connectors.md) - connector notes and integration direction.
- [Codex Migration](CODEX.md) - mapping from GitHub Copilot customization to Codex instructions, agents, and skills.

## Document Flow

```mermaid
flowchart TD
	ROOT[docs/] --> ARCH[ARCHITECTURE.md]
	ROOT --> PROD[PRODUCTION_READINESS.md]
	ROOT --> MCP[mcp-connectors.md]
	ROOT --> CODEX[CODEX.md]
	ARCH --> STAGES[Stage and contract understanding]
	PROD --> GAPS[Readiness gaps and acceptance criteria]
	MCP --> SOURCES[Connector behavior and direction]
	CODEX --> CUSTOM[Codex customization routing]
```

## Recent Updates

- [Codex Migration](CODEX.md) documents the `.github` to `.codex` mirror, root `AGENTS.md`, and validation commands for migrated skills.
- [MCP Connectors](mcp-connectors.md) now documents the Google Drive connector (Phase 1): OAuth/service-account auth, folder scan, Docs/Sheets/Slides export, stable replay event IDs, and the `/googledrive/status` and `/googledrive/ingest` API endpoints.

## Maintenance Checklist

- Update this index when new top-level documentation is added.
- Keep document names and links aligned after renames.
- Reflect major architecture or readiness changes in the linked docs.
