# Frontend App

SvelteKit application surface for ContextOS — chat-first, project-scoped delivery intelligence.

## Routes

| Route | Purpose |
|---|---|
| `/` | **Chat-first homepage.** Conversational interface for delivery analysis. Sidebar shows project identity, connector knowledge state, and Codex CLI status. |
| `/connectors` | Connector debug surface (preserved from previous homepage). Useful for testing individual connector ingestion flows. |
| `/findings` | Advanced findings viewer. Role-based PMO/presentation/service/QA/architecture findings with mismatch detail. |

## Chat-First Workflow

On first visit:
1. The **Knowledge Installation Wizard** opens automatically (`+ Install Knowledge` button).
2. Select the connectors you want to enable — GitHub, Jira, Slack, Notion, SharePoint/OneDrive, Google Drive, or Filesystem.
3. Enter the URI for each connector (repo, channel, project URL, etc.).
4. Click **Install Knowledge** — each connector is ingested sequentially with live progress streaming.
5. Once complete, chat with your project. Try asking:
   - `show findings`
   - `status`
   - `help`

## Project Identity

Projects are keyed by **workspace folder path** (stored in `localStorage`). Click the project name in the sidebar to change the path. Each workspace gets its own chat history and knowledge state.

## Chat Commands

| Command | Behavior |
|---|---|
| `show findings` | Run analysis and show mismatches for latest ingested connector |
| `status` | Show connector and Codex plugin readiness |
| `install knowledge` | Open the Knowledge Installation Wizard |
| `clear` | Clear chat history for current project |
| `connectors` | Navigate to connector debug page |
| `help` | Show command reference |

Natural language is also routed — questions about mismatches, delivery gaps, or status are matched to the appropriate backend call.

## Initial Knowledge Installment

The `KnowledgeInstall` component (`src/lib/components/knowledge/KnowledgeInstall.svelte`) is the core first-run flow:
- Shows readiness per connector (Codex plugin installed + logged in).
- Runs `runConnectorIngest` for each enabled + configured connector sequentially.
- Streams live progress logs per connector.
- Marks the project as knowledge-ready on completion.
- Reopenable at any time via the sidebar button.

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
    CHAT[/ chat homepage] --> STORE[projectStore.ts]
    CHAT --> CMDS[Chat command router]
    CMDS --> INGEST[runConnectorIngest]
    CMDS --> FINDINGS[postFindings]
    CMDS --> STATUS[checkCodexStatus]
    INGEST --> API[api.ts]
    FINDINGS --> API
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


## Routes

- `/` — connector and ingestion debug workspace.
- `/findings` — graph-backed findings UI with role-specific summaries, PMO view model, and assistive execution metadata.

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

The canonical test patterns live in [frontend-jest-swc-patterns](../../.github/skills/frontend-jest-swc-patterns/) and apply to `src/lib/__tests__/*.test.ts`.
