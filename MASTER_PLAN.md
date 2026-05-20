# ContextOS — Master Execution Plan

> Personal local-first AI context synchronization platform

---

# 1. Ultimate Goal

Build a local-first modular AI operating system that:

- synchronizes organizational understanding
- extracts business logic automatically
- detects FE / BE / PMO mismatch
- builds persistent context graph
- orchestrates hidden AI execution
- continuously updates delivery intelligence

---

# 2. Development Philosophy

## IMPORTANT

Do NOT build:
- huge SaaS
- multi-tenant architecture
- enterprise auth
- complicated infra

Build ONLY:
- local-first
- personal-use focused
- modular domains
- composable pipelines

---

# 3. MVP Goal

FIRST SUCCESS METRIC:

```text
Detect real FE / BE misunderstanding automatically.
```

NOT:

* autonomous PM
* AGI project management
* full AI replacement

---

# 4. Domain Architecture

---

# Domain 1 — Source

## Responsibility

Receive external data.

## Examples

* Slack
* Jira
* GitHub
* OpenAPI
* Excel
* Local Files

## Output

Raw ingestion events.

---

# Domain 2 — Ingestion

## Responsibility

Convert all source data into universal ingestion events.

## Output Example

```json
{
  "source": "slack",
  "type": "message",
  "content": "...",
  "metadata": {}
}
```

---

# Domain 3 — Normalization

## Responsibility

Convert all inputs into normalized documents.

## Goal

Unified processing format.

---

# Domain 4 — Classification

## Responsibility

Classify:

* business logic
* API discussion
* PMO risk
* FE concern
* BE concern
* blocker
* decision

---

# Domain 5 — Extraction

## Responsibility

Extract structured entities:

* API fields
* DB columns
* enums
* requirements
* services
* dependencies

---

# Domain 6 — Identity Resolution (CORE DOMAIN)

## Responsibility

Resolve:

* same meaning
* different naming
* different systems

into:

```text
single canonical identity
```

## Example

```text
refund_status
refundState
返金状態
```

→ same entity

---

# Domain 7 — Relationship

## Responsibility

Build graph relationships.

Example:

```text
Requirement
→ affects API
→ affects DB
→ affects FE screen
```

---

# Domain 8 — Context Graph

## Responsibility

Persistent organizational memory.

Stores:

* entities
* aliases
* relationships
* history

---

# Domain 9 — Reasoning

## Responsibility

Analyze graph:

* mismatch detection
* impact analysis
* PMO visibility
* implementation gaps

---

# Domain 10 — Execution

## Responsibility

Hidden AI orchestration.

Internally:

* launch Codex CLI
* analyze repo
* generate implementation understanding

---

# Domain 11 — Presentation

## Responsibility

Role-based outputs.

Views:

* PMO
* FE
* BE
* QA
* Architecture

---

# 5. Folder Structure

```text
contextos/

├── apps/
│
│   ├── frontend/
│   │
│   ├── api/
│   │
│   └── ai-worker/
│
├── internal/
│
│   ├── source/
│   │   ├── slack/
│   │   ├── jira/
│   │   ├── github/
│   │   ├── openapi/
│   │   ├── excel/
│   │   └── filesystem/
│   │
│   ├── ingestion/
│   │
│   ├── normalization/
│   │
│   ├── classification/
│   │
│   ├── extraction/
│   │
│   ├── identity/
│   │
│   ├── relationship/
│   │
│   ├── graph/
│   │
│   ├── reasoning/
│   │
│   ├── execution/
│   │
│   └── presentation/
│
├── domain/
│
│   ├── entities/
│   │
│   ├── contracts/
│   │
│   ├── pipelines/
│   │
│   ├── events/
│   │
│   └── types/
│
├── storage/
│
│   ├── raw/
│   │
│   ├── parsed/
│   │
│   ├── embeddings/
│   │
│   └── snapshots/
│
├── prompts/
│
├── docker/
│
├── scripts/
│
├── migrations/
│
├── docs/
│
└── tests/
```

---

# 6. Event-Driven Architecture

Everything should emit events.

Examples:

```text
document.ingested
document.normalized
entity.extracted
identity.resolved
relationship.created
mismatch.detected
codex.analysis.completed
```

---

# 7. Identity Resolution Strategy

MOST IMPORTANT ENGINE.

## Multi-layer resolution

### Layer 1 — Exact Match

```text
refund_status == refund_status
```

### Layer 2 — Semantic Similarity

Embedding similarity.

### Layer 3 — Relationship Similarity

Used by same API/service/screen.

### Layer 4 — Historical Merge Memory

Track renamed entities.

### Layer 5 — Human Confirmation

Critical merges require approval.

---

# 8. Storage Strategy

## PostgreSQL

Structured truth.

## pgvector

Semantic retrieval.

## Filesystem

Raw snapshots.

---

# 9. AI Strategy

AI should:

* classify
* extract
* summarize
* reason

AI should NOT:

* become source of truth
* directly mutate graph blindly

---

# 10. Execution Strategy

User never directly interacts with Codex.

System internally:

* prepares context
* launches Codex CLI
* imports analysis
* updates graph

---

# 11. MVP Build Order

## Phase 1

* repo scanner
* Slack export parser
* OpenAPI parser
* normalization pipeline

## Phase 2

* extraction engine
* identity resolution
* relationship graph

## Phase 3

* mismatch detection
* PMO summaries
* FE / BE synchronization

## Phase 4

* hidden Codex execution
* implementation analysis

---

# 12. First Killer Feature

```text
Business Logic Synchronization Engine
```

Detect:

* outdated FE assumptions
* missing API fields
* inconsistent enums
* unresolved business logic

---

# 13. Long-Term Moat

The moat is NOT:

* prompts
* connectors
* models

The moat IS:

* context graph
* organizational memory
* identity resolution
* relationship intelligence
* synchronized understanding

---

# 14. Final Rule

Always optimize for:

```text
understanding synchronization
```

NOT:

```text
AI hype
```
