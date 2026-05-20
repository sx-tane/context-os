# Issue 20: Validate first killer feature

## Description

End-to-end validation for the Business Logic Synchronization Engine: ingest sources, resolve identities, build graph, and detect real FE/BE mismatch.

## Acceptance criteria

- Test covers ingestion through an MCP connector.
- Test runs normalization, classification, extraction, identity resolution, relationship graph construction, and mismatch detection.
- Test verifies the graph contains extracted entities and reports at least one mismatch.
