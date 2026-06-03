# Frontend App

SvelteKit application surface for ContextOS — local-first, workspace-scoped delivery intelligence.

## Routes

| Route | Purpose |
|---|---|
| `/` | **Single-window product workspace.** Left rail lists workspaces and sources, center surface handles dashboard/search/chat commands, right truth panel shows answer, evidence, analysis, and graph. |
| `/connectors` | Connector debug surface, preserved by direct URL but not linked from the main product window. |
| `/findings` | Advanced findings viewer, preserved by direct URL but not linked from the main product window. |

## Product Window Workflow

On first visit:
1. Pick or add a workspace from the left rail. Workspace state is local and path-scoped.
2. Open **ADD SOURCE** to configure GitHub, Jira, Slack, Notion, SharePoint/OneDrive, Google Drive, or Filesystem.
3. Source setup appears inline inside the product window instead of as a blocking modal.
4. Ask local source questions in the command bar, for example `give me today slack messages`, `recent jira tickets`, or `latest drive docs`.
5. The right truth panel shows the local answer, cited artifacts, analysis findings, and graph entities.

## Project Identity

Projects are keyed by **workspace folder path** (stored in `localStorage` and mirrored to the API workspace table). The home screen can list, add, and switch multiple workspaces. Each workspace gets its own chat history and connector knowledge state.

## Chat Commands

| Command | Behavior |
|---|---|
| source question | Calls `POST /chat/query` and answers from local ingested artifacts only |
| `show findings` | Runs analysis and shows mismatches for the latest ready connector |
| `status` | Routed through local chat status handling |
| `install knowledge` / `add source` | Opens the inline source setup panel |
| `clear` | Clear chat history for current project |

Natural language source questions do not fall back to findings. If no matching local artifact exists, the answer says no local data was found.

## Initial Knowledge Installment

The `KnowledgeInstall` component (`src/lib/components/knowledge/KnowledgeInstall.svelte`) is the core first-run flow:
- Shows readiness per connector (Codex plugin installed + logged in).
- Runs `runConnectorIngest` for each enabled + configured connector sequentially.
- Streams live progress logs per connector.
- Marks the project as knowledge-ready on completion.
- Reopenable at any time via the sidebar button.

## Design Rules

Svelte UI work follows the Codex frontend design skill at [`../../.codex/skills/contextos-frontend-design/`](../../.codex/skills/contextos-frontend-design/).
Use the current restrained mono theme: warm background, separator-based rows, padded underline-fill controls, local hidden-scroll panes where needed, and readable graph/source/chat surfaces without extra boxed noise.

## Project Store

`src/lib/projectStore.ts` — Svelte writable store persisted to `localStorage`:
- `project` — project metadata, connectors, knowledge install timestamp.
- `chatMessages` — chat history (last 200 messages).
- `openProject(path)` — switch workspace.
- `setConnectorKnowledge(connector, uri, status)` — update connector state.
- `addMessage / replaceMessage` — chat message management.
- `markKnowledgeInstalled()` — stamp knowledge ready timestamp.

## Frontend Flow

```mermaid
flowchart TD
    CHAT[/ product workspace] --> STORE[projectStore.ts]
    CHAT --> CMDS[Chat command router]
    CMDS --> QUERY[postChatQuery]
    CMDS --> INGEST[runConnectorIngest]
    CMDS --> FINDINGS[postFindings]
    CHAT --> GRAPH[getGraphData]
    INGEST --> API[api.ts]
    QUERY --> API
    FINDINGS --> API
    GRAPH --> API
    STORE --> LS[(localStorage)]
    KI[KnowledgeInstall.svelte] --> INGEST
    API --> BACKEND[Go API]
    CONNECTORS[/connectors debug page] --> API
    FINDINGS_PAGE[/findings page] --> API
```

## Connector Support

All seven connectors are available in the knowledge install wizard:

| Connector | Codex Plugin | Auth Fallback |
|---|---|---|
| GitHub | `github@openai-curated` | `GITHUB_TOKEN` env var |
| Jira | `atlassian-rovo@openai-curated` | `JIRA_TOKEN` + `JIRA_EMAIL` + `JIRA_BASE_URL` |
| Slack | `slack@openai-curated` | `SLACK_BOT_TOKEN` env var |
| Notion | `notion@openai-curated` | `NOTION_TOKEN` env var |
| SharePoint / OneDrive | `sharepoint@openai-curated` | `SHAREPOINT_ACCESS_TOKEN` or client credentials |
| Google Drive | `google-drive@openai-curated` | OAuth credentials or service account |
| Filesystem | — (direct) | Server-visible path |

Connectors are **Codex-only by default**. The wizard shows a warning and disables the connector toggle if the Codex plugin is not installed or Codex CLI is not logged in.

## Filesystem Supported Formats

| Format | Extensions | Extraction |
|---|---|---|
| Folder | Directory path | Recursive child-file events |
| Text and Markdown | `.txt`, `.md` | Read directly |
| Code and config | `.go`, `.ts`, `.json`, `.yaml`, `.toml`, `.sql` | OpenAPI JSON/YAML gets endpoint/schema metadata |
| Spreadsheet | `.xlsx`, `.csv` | Cell, sheet, row, value, formula facts |
| Word document | `.docx` | Paragraph text |
| PDF | `.pdf` | Best-effort page text |
| PowerPoint | `.pptx` | Slide text |

## Type generation

TypeScript types for API responses and events are auto-generated from the OpenAPI spec. Do **not** edit `src/lib/generated/api.d.ts` by hand.

Refresh types after any API shape change:

```bash
cd apps/frontend
bun run codegen
```

This reads `apps/api/_docs/swagger.json` and writes `src/lib/generated/api.d.ts`. The generated file is committed to the repository because TypeScript needs it to compile. `start-all.sh` runs this step automatically on every startup.

```
swag init  →  apps/api/_docs/swagger.json  →  bun run codegen  →  src/lib/generated/api.d.ts
```

Frontend-specific types that have no swagger equivalent (`IngestRequest`, `SourceConnectorConfig`, `ConnectorKind`, etc.) remain in `src/lib/types.ts` and are maintained manually.

## Testing

Frontend utility tests run with Jest and SWC for fast TypeScript compilation.
The Jest runtime is configured in `jest.config.cjs`, including SWC transformation and `$lib` module mapping.

```bash
cd apps/frontend
bun run test
```

Use coverage when changing shared API helpers, ingest runners, reauth runners, or other `src/lib/` utilities:

```bash
cd apps/frontend
bun run test:coverage
```

The canonical test patterns live in [frontend-jest-swc-patterns](../../.codex/skills/frontend-jest-swc-patterns/) and apply to `src/lib/__tests__/*.test.ts`.
