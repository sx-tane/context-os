---
description: "Use when creating or changing source connectors for GitHub, Jira, Slack, filesystem, or filesystem-supported OpenAPI/spreadsheet formats. Covers idempotent ingestion and replay-safe behavior."
applyTo: "internal/source/**/*.go"
---

# Connector Instructions

## Skill

For a full step-by-step guide, skeletons, and a completion checklist, apply the **contextos-api-handler** skill.

## Connector Shape

Every connector must:

- Wrap `source.NewMCPConnector("<name>", contracts.Capability<X>)` — do not implement the interface from scratch.
- Export only `NewConnector() contracts.MCPSourceConnector` as the public constructor.
- Implement `Name()` and `Capabilities()` by delegating to the base connector.
- Clone metadata with `cloneMetadata` before mutating: `req.Metadata = cloneMetadata(req.Metadata)`.
- Return `contracts.ConnectorError` (not bare `fmt.Errorf`) for domain-level failures.

## Reliability Rules

- Same `URI` + `Content` must produce the same `event.ID` (idempotency guaranteed by `events.New`).
- Record the source metadata required for replay and audit (`source_uri`, `source_cursor`, connector name).
- Fail with actionable errors that include connector name, object type, and object identifier.
- Add tests for duplicate ingestion, context cancellation, and missing required fields.
- Update `internal/source/README.md` or the connector README when capabilities, metadata, replay behavior, or setup changes.
