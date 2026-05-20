# Issue 3: Build MCP source connector interface

## Description

Define one common MCP connector contract for all ContextOS source integrations.

## Acceptance criteria

- All connectors expose name, capabilities, and ingest behavior.
- Connector ingest output is `document.ingested` events.
- The contract is source-agnostic and supports GitHub, Slack, Jira, OpenAPI, Excel, and filesystem connectors.
