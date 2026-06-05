# Internal Chat

Chat service for answering workspace-scoped questions from persisted ContextOS repositories and optional Codex-backed live source context.

## Files

| File | Purpose |
| --- | --- |
| `answer_sections.go` | Parses structured live Codex JSON into backward-compatible answer text plus source-card sections. |
| `chat.go` | Classifies local chat intent, resolves workspace scope, queries artifacts and sync state, resets live session metadata, and builds answer summaries. |
| `codex_answerer.go` | Runs live Codex chat with workspace-and-connector scoped `codex exec` session metadata and `codex exec resume` on later turns. |
| `chat_test.go` | Verifies intent routing, workspace resolution, time range inference, and answer construction. |

## Behavior

The service supports artifact, status, findings, and unsupported intents. It always resolves workspace scope and lists connector sync state before answering. Explicit request fields from the frontend take precedence over message inference, so a concrete route such as `connector: "jira"` and `source_uri: "BKGDEV-8466"` remains issue-scoped even when the message is ambiguous. For plugin-backed connectors (`github`, `jira`, `slack`, `notion`, `googledrive`, `sharepoint`), source questions use a `LiveAnswerer` first when the message includes a source link or can be matched to a saved `connector_syncs` source. When a meaningful prompt has no explicit connector or source, the service starts every eligible connected source lookup concurrently and returns sections in stable connector order: Jira, GitHub, Slack, Google Drive, Notion, SharePoint. The plain chat answer synthesizes those sections into meaning, behavior, change/status, and open-item lines before naming evidence-source counts; the structured `answer_sections` remain the detailed per-source cards. Connector-level scope rows such as `github` are used only when that connector has no more concrete saved source. Failed parallel lookups are retried serially once before local fallback so Codex CLI startup contention does not erase live answers. Language-only, status, and findings prompts do not fan out. Filesystem questions remain local-first because filesystem content is ingested into ContextOS storage.

If live Codex lookup fails, the answer names the live failure and then falls back to local artifacts when available. Callers can provide `Query.Progress` to receive Codex-style transcript lines while the live lookup runs, including the plugin/source being checked, CLI startup/resume, summarized Codex JSONL activity events, heartbeat status from the API layer, and completion/failure notes. `Query.ResponseLanguage` is normalized against the actual message before it reaches the live prompt or deterministic local answer builders. This keeps English-heavy mixed prompts such as `kkg payment 決済GW linkedFlag` in English even if a client mistakenly sends `zh`, while real Chinese/Japanese/Korean prompts keep their language. Structured Codex JSON is parsed into `Result.AnswerSections` while `Result.Answer` remains a plain text summary for compatibility. The API chat handler persists concrete live answer sections as local evidence after successful Codex answers; this service does not write evidence directly. Local artifacts, graph output, findings, evidence, and confidence remain the auditable source of truth for double-checking and analysis.

Live Codex chat keeps one CLI conversation per workspace connector. The first live turn runs `codex exec --sandbox read-only --json --color never -o <tmp> -`, writes the prompt to stdin, parses the `session_meta.payload.id` JSONL event, and stores that ID under `storage/codex-chat-sessions/<workspace-id>_<connector>.json`. Later turns for the same workspace connector run `codex exec resume --json -o <tmp> <session-id> -` and again provide the prompt through stdin. Calls are serialized per workspace connector so two requests cannot race the same conversation, while broad multi-connector fanout can run separate connector sessions in parallel. If the stored session is missing, archived, or unreadable, the answerer deletes the local pointer, starts a fresh session, and stores the new ID. `Service.ResetSession` deletes local session pointers for the workspace, including connector-scoped pointers; it does not archive or delete Codex global session files.

GitHub source questions infer the configured repository source from sync state when the user names only a repo slug such as `tourii-backend`. This keeps answers scoped to the requested repo instead of falling back to every GitHub artifact in the workspace. Pasted Jira, GitHub, Slack, Notion, Google Drive, and SharePoint links can route to the matching live plugin even when the exact source was not saved during setup. Live Codex prompts require plugin-only read behavior: GitHub chat must use the GitHub Codex plugin/context and must not fall back to `gh`, public web search, or local repository remotes. Live Codex prompts ask for a direct answer first and compact JSON source sections with capped facts, so saved evidence remains useful in Activity and graph views without becoming a long inventory.

```mermaid
flowchart TD
  Q[Chat query] --> W[Resolve workspace]
  W --> S[List sync state]
  S --> M[Resolve source URI from link or saved source]
  M --> X{Explicit source?}
  X -->|no| Z[Start connected live scopes in parallel]
  X -->|yes| I[Classify intent]
  Z --> R
  M --> I[Classify intent]
  I -->|plugin source| C[Ask Codex LiveAnswerer]
  C --> N{Stored session?}
  N -->|yes| V[Codex exec resume]
  N -->|no| E[Codex exec new session]
  V -->|success| R[Live answer]
  E -->|success| R
  V -->|missing session| E
  V -->|failure| A[Query local artifacts]
  E -->|failure| A
  I -->|filesystem/local source| A
  A --> R
  I -->|status| T[Summarize syncs]
  I -->|findings| F[Point to findings workflow]
  T --> R
  F --> R
```

## Maintenance Notes

- Keep filesystem answers deterministic and local-first.
- Keep live Codex answers labeled through `Result.Provider`.
- Preserve workspace scoping before querying artifacts or sync state.
- Update `apps/api/handler/chat/README.md` when service result fields change.
