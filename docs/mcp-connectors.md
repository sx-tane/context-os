# MCP Connector Architecture

All external source integrations in ContextOS are MCP-first connectors. Each connector implements the shared `MCPSourceConnector` contract and converts source-specific input into `document.ingested` events.

## Required connectors

- GitHub MCP connector
- Slack MCP connector
- Jira MCP connector
- OpenAPI MCP connector
- Excel MCP connector
- Filesystem MCP connector

## Connector output

Each connector emits raw ingestion events that are then normalized, classified, extracted, resolved, related, stored in the context graph, and analyzed for delivery mismatches.
