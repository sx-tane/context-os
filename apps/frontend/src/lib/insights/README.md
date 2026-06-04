# insights

Pure helpers for the homepage insight surface. Keep cross-tab freshness and status derivation here so `+page.svelte`, `FindingsView`, `GraphView`, and `ActivityView` do not each invent their own source, Activity, Graph, or Findings state.

## status.ts

`buildInsightStatus` derives the shared model used by the insight status strip and footer:

- concrete analysis-ready source count vs chat-only live connector scopes;
- latest Activity evidence timestamp and event count;
- Graph node/link counts plus an empty/ready/waiting label;
- last manual Findings analysis freshness;
- Findings state: `not_run`, `current`, `stale`, or `no_concrete_sources`.

Findings remain manual. Chat and evidence saves may refresh Activity and Graph immediately, but the status helper marks Findings stale until the user runs analysis again.
