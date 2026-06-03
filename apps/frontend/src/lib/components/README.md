# components

Svelte UI components for the ContextOS frontend. Organised into three sub-directories by responsibility.

```
components/
├── connectors/   Connector-specific forms and cards (GitHub, Jira, Slack, generic SourceConnector)
├── feedback/     Status display, log streams, error messages, and ingest results
└── ui/           Primitive building blocks (Button, FormField, ModeToggle)
```

## Sub-directory overview

| Directory                             | What lives there                                                                                       |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| [`connectors/`](connectors/README.md) | `ConnectorCard`, `SourceConnector`, `GitHubConnector`, `JiraConnector`, `SlackConnector`, `CodexBadge` |
| [`feedback/`](feedback/README.md)     | `StatusSection`, `LogPanel`, `IngestResult`, `ErrorPanel`                                              |
| [`ui/`](ui/README.md)                 | `Button`, `FormField`, `ModeToggle`                                                                    |

## Data flow

```
+page.svelte
  └─ ConnectorCard / GitHubConnector / SourceConnector   (input & trigger)
       └─ ingestRunner / reauthRunner                     (orchestration)
            └─ api.ts                                     (network)
  └─ StatusSection                                        (health probes)
  └─ LogPanel                                             (SSE log stream)
  └─ IngestResult                                         (parsed events)
  └─ ErrorPanel                                           (error display)
```

## Design Pattern

Component visual changes should apply the [`contextos-frontend-design`](../../../../.codex/skills/contextos-frontend-design/) skill. Keep components aligned with the current app: warm neutral surfaces, separator-based rows, explicit left/right padding, padded underline-fill buttons, and local scroll behavior that avoids distracting nested visible scrollbars.
