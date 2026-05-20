---
description: "Use when creating or changing source connectors for GitHub, Jira, Slack, OpenAPI, Excel, or filesystem. Covers idempotent ingestion and replay-safe behavior."
applyTo: "internal/source/**/*.go"
---
# Connector Reliability Instruction

- Preserve immutable raw payload capture.
- Emit stable event keys for deduplication.
- Handle retries without creating duplicate ingestion records.
- Record source metadata required for replay and audit.
- Fail with actionable errors that include source, object type, and object identifier.
- Add tests for duplicate events, out-of-order events, and partial failures.
