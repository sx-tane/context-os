# components

Svelte UI components for the ContextOS frontend. Organised into sub-directories by responsibility.

```
components/
├── ConfirmModal.svelte
├── chat/         Homepage and reusable chat thread surfaces
├── connectors/   Connector-specific forms and cards (GitHub, Jira, Slack, generic SourceConnector)
├── feedback/     Status display, log streams, error messages, and ingest results
├── insights/     Homepage insight tab views and source/workspace summary
└── ui/           Primitive building blocks (Button, FormField, ModeToggle)
```

## Sub-directory overview

| Directory                             | What lives there                                                                                       |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| [`ConfirmModal.svelte`](ConfirmModal.svelte) | Shared destructive-action confirmation dialog used by workspace removal and source reset flows. |
| [`chat/`](chat/README.md)             | `ChatPanel`, `ChatInput`, `ChatMessage`, `ChatThread`                                                  |
| [`connectors/`](connectors/README.md) | `ConnectorCard`, `SourceConnector`, `GitHubConnector`, `JiraConnector`, `SlackConnector`, `CodexBadge` |
| [`feedback/`](feedback/README.md)     | `StatusSection`, `LogPanel`, `IngestResult`, `ErrorPanel`                                              |
| [`insights/`](insights/README.md)     | `FindingsView`, `GraphView`, `ActivityView`, `WorkspaceSummary`                                        |
| [`ui/`](ui/README.md)                 | `Button`, `FormField`, `ModeToggle`                                                                    |

## Data flow

```
+page.svelte
  └─ ConnectorCard / GitHubConnector / SourceConnector   (input & trigger)
       └─ ingestRunner / reauthRunner                     (orchestration)
            └─ api.ts                                     (network)
  └─ StatusSection                                        (health probes)
  └─ WorkspaceSummary / FindingsView / GraphView / ActivityView
                                                           (homepage right pane)
  └─ ConfirmModal                                         (destructive confirmations)
  └─ LogPanel                                             (SSE log stream)
  └─ IngestResult                                         (parsed events)
  └─ ErrorPanel                                           (error display)
```

## Design Pattern

Component visual changes should apply the [`contextos-frontend-design`](../../../../../.codex/skills/contextos-frontend-design/) skill. Keep components aligned with the current app: warm neutral surfaces, separator-based rows, explicit left/right padding, padded underline-fill buttons, and local scroll behavior that avoids distracting nested visible scrollbars.
