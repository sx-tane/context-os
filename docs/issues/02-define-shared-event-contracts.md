# Issue 2: Define shared event contracts

## Description

Add shared event types for the ContextOS event-driven pipeline.

## Acceptance criteria

- Define `document.ingested`, `document.normalized`, `entity.extracted`, `identity.resolved`, `relationship.created`, and `mismatch.detected`.
- Provide a reusable event envelope with source, subject, content, metadata, and timestamp fields.
