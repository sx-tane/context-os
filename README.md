# ContextOS

Local-first AI operating system for delivery intelligence and organizational context synchronization.

ContextOS continuously converts fragmented delivery signals across engineering and business tools into a structured, queryable, and explainable understanding of how work is actually being delivered.

## Why ContextOS Exists

Most organizations do not fail because information is missing. They fail because understanding is fragmented.

Typical failure patterns:

- presentation and service layers implement different business assumptions
- PMO status does not match engineering reality
- Jira, Slack, GitHub, docs, and spreadsheets disagree on scope and intent
- key concepts are renamed across teams and languages
- repeated clarification work consumes delivery time

ContextOS addresses this by synchronizing meaning, not just storing data.

## Product Thesis

The hardest part of delivery intelligence is not generation. It is identity resolution under ambiguity.

The same concept often appears in many forms:

```text
refund_status
refundState
refund flag
返金状態
```

ContextOS resolves these into shared business entities, links them to implementation artifacts, and surfaces misalignment before it becomes delay or defect.

## What ContextOS Is

- organizational memory layer
- delivery intelligence engine
- context synchronization platform
- business logic understanding system

ContextOS is not a generic chatbot, coding assistant, or issue tracker replacement.

## System Architecture

The pipeline processes source events through eleven deterministic stages from ingestion to presentation. For domain diagrams, stage contracts, dependency rules, and per-stage implementation guides, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### Runtime Component Architecture

```mermaid
flowchart TD
  UI[Context UI: SvelteKit]
  API[API Service: Go]
  AW[AI Worker: Python]
  Q[NATS]
  DB[(PostgreSQL + pgvector)]
  ST[Storage: raw/parsed/snapshots]
  SRC[External Sources]

  UI --> API
  API --> DB
  API --> Q
  API --> ST
  API --> SRC

  Q --> AW
  AW --> DB
  AW --> ST
```

## Storage Model

Data is persisted across processing maturity levels:

- raw: immutable source payloads
- parsed: normalized and extracted structured records
- snapshots: reproducible point-in-time context states
- embeddings: vectorized semantic representations

Domain contracts and package structure are documented in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Tech Stack

- context UI: SvelteKit
- APIs and core orchestration: Go
- AI/LLM task workers: Python
- database and vector storage: PostgreSQL + pgvector
- async messaging: NATS
- search: PostgreSQL full text (OpenSearch optional later)
- AI execution strategy: provider-agnostic, hidden internal execution interfaces

## Getting Started (Local)

### 1) Set up prerequisites

```bash
./scripts/setup-local.sh
```

Installs Go, Bun, Python 3.12, `uv`, Codex CLI, and the GitHub, Atlassian Rovo, and Slack Codex plugins on Linux. Run once on a fresh machine.

### 2) Validate the repository

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g apps/api/main.go -o apps/api/docs
go mod tidy
go test ./...
```

API docs are generated locally under `apps/api/docs/` and are ignored by git. `scripts/start-all.sh` regenerates them automatically when `swag` is installed.

### 3) Start all services

```bash
./scripts/start-all.sh
```

Starts the API, context UI dev server, and AI worker together. Press `Ctrl+C` to stop all processes. If `uv` is not found, the AI worker is skipped automatically.

Local connector UI is served from `apps/frontend` and includes GitHub, Jira, Slack, filesystem, and Codex CLI status/login flows. Filesystem ingestion covers single files, recursive local folders, spreadsheets, and OpenAPI spec files.

## Production Delivery Plan

The plan below targets production-grade organizational intelligence with local-first operation, replay-safe ingestion, durable graph memory, and explainable misalignment findings.

### Phase 0: Platform Foundation

Goals:

- define canonical domain contracts and event envelopes
- establish ingestion idempotency and replay safety
- set up local-first developer workflow and baseline observability

Exit criteria:

- connectors can ingest repeatably without duplication
- each pipeline stage has traceable input/output identifiers
- baseline metrics exist for throughput, latency, and failure rate

### Phase 1: Source Reliability and Parsing Quality

Goals:

- production-ready connectors for GitHub, Slack, Jira, and filesystem
- robust parsing for code, tickets, discussions, OpenAPI specs, spreadsheets, and documents
- snapshot versioning for reproducible analysis

Exit criteria:

- end-to-end sync runs across all core connectors and supported filesystem formats
- parse coverage and error rates are measurable and within target
- snapshots can reconstruct a prior context state deterministically

### Phase 2: Identity Resolution Engine

Goals:

- alias dictionary + embedding-assisted identity candidate generation
- confidence scoring, merge rules, and conflict handling
- multilingual and naming-convention-aware matching

Exit criteria:

- canonical entity linking reaches agreed precision/recall targets
- conflicts are surfaced with explainable reasons
- manual correction loop exists and updates future resolution behavior

### Phase 3: Relationship Graph and Dependency Intelligence

Goals:

- model cross-artifact relationships (feature, API, owner, risk, timeline)
- detect dependency chains and ownership bottlenecks
- support graph queries for impact analysis

Exit criteria:

- graph supports critical queries for planning and incident triage
- dependency risk scoring is available through API outputs
- relationship provenance is visible for auditability

### Phase 4: Reasoning and Misalignment Detection

Goals:

- detect cross-layer context drift, PMO vs implementation drift, and requirement gaps
- generate explainable findings with evidence links
- prioritize risks by likely delivery impact

Exit criteria:

- findings include confidence, impact, and evidence references
- false-positive rate is tracked and controlled
- recommendation quality is validated with team feedback

### Phase 5: Operational Intelligence Productization

Goals:

- delivery intelligence dashboards and periodic summaries
- role-specific views (engineering, PMO, leadership)
- notification and workflow hooks for actionability

Exit criteria:

- users can move from insight to action in one workflow
- recurring reports are stable and trusted by delivery stakeholders
- usage and outcome metrics show measurable planning improvement

### Phase 6: Scale, Governance, and Ecosystem

Goals:

- tenancy, access control, retention, and compliance controls
- plugin-based connector and rule ecosystem
- continuous evaluation framework for model behavior and regressions

Exit criteria:

- governance controls satisfy organizational security requirements
- extension points are documented and stable
- evaluation suite blocks regressions before release

## Success Metrics

ContextOS should be judged by delivery outcomes, not model novelty.

- reduction in cross-layer misalignment incidents
- reduction in repeated clarification cycles
- improved predictability of delivery milestones
- faster impact analysis during requirement changes
- higher confidence in cross-team status accuracy

## Current Repository Structure

- apps: deployable surfaces (api, context UI, ai-worker)
- internal: domain implementations and orchestration logic
- domain: cross-domain contracts, entities, events, and pipeline types
- storage: local-first data layers by processing stage
- tests: pipeline-level validation and integration checks
- docs: architecture and connector specifications

## Near-Term Build Priorities

1. finalize canonical contracts in domain and internal boundaries
2. implement connector reliability guarantees (idempotency + replay)
3. establish measurable identity-resolution benchmarks
4. ship first misalignment reports with explainable evidence
5. add role-specific presentation outputs for PMO and engineering

## Design Principles

- local-first by default
- modular domain boundaries
- observable and explainable intelligence
- deterministic pipelines where possible
- provider-agnostic AI execution layer
- replaceable connectors and reasoning strategies

## Long-Term Outcome

Build persistent organizational intelligence that survives personnel change, tool churn, and naming drift while continuously improving delivery alignment across business and engineering.
