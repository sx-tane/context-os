# chat components

Svelte components for the main local chat thread.

## Components

| File | Purpose |
| --- | --- |
| `ChatPanel.svelte` | Renders the homepage report-agent pane, compact stream progress, safe Markdown text, structured source cards, source traces, message evidence previews, findings previews, and composer. |
| `ChatInput.svelte` | Captures and submits the user's chat message. |
| `ChatMessage.svelte` | Renders one chat message and its metadata. |
| `ChatThread.svelte` | Renders the message list and scrolls to the latest message after updates. |

## Usage

These components are presentation-only. API calls, workspace state, and message orchestration should stay in the route or shared frontend helpers such as `$lib/chat/controller`. Query answer metadata renders `ChatQueryResult.provider` so users can distinguish live Codex lookup from local DB evidence.

Pending query progress lives in `ChatMessage.stream` instead of the answer text. `ChatPanel.svelte` renders it as a collapsed stream block by default: running and completed streams show status, the latest Codex-style `›` or `•` progress line, and the final summary when available. Expanding the block shows the full stream transcript in order.

The composer is a compact multiline textarea. Enter and the send button submit, while Shift+Enter inserts a new line for longer source questions or pasted snippets. The textarea has a capped height with local scrolling so large prompts do not resize the whole chat pane. `ChatPanel.svelte` accepts synchronous or async clear handlers so the route can clear browser chat and best-effort backend chat session state from one control.

Query and analysis result messages render summary-first: the chat text starts with `**Summary**`, then includes `**Answer**` only when the answer adds detail beyond the summary. When `chatResult.answer_sections` is present, `ChatPanel.svelte` renders one expanded source section per answer section with connector/source label, source link, summary, facts, open items, coding notes, and links. Source summaries and list items use the shared safe Markdown block from `components/ui`, so headings, bullets, numbered lines, URLs, inline code, and bold spans remain readable without raw HTML. Expanded source sections stay transparent and use restrained connector-colored titles to match Activity. Each concrete source section may expose **Ask** and **Pin** actions supplied by the route: Ask only prefills the composer with the selected `connector:source_uri` context, and Pin adds the source to the durable analysis basket. The panel then shows a lightweight Local DB save status when the backend reports `evidence_save_status`, followed by a source trace with provider, connector, source URI, non-duplicative stream summary, and artifact count. Artifact evidence details remain available when `chatResult.artifacts.length > 0`. Live Codex answers without artifacts are labeled as a live source trace; a separate `Local DB: saved ...; graph updated`, `skipped ...`, or `save failed ...` line tells the user whether the live answer evidence was persisted and whether Graph updated. Findings are not refreshed from chat saves.

Chat message text preserves Japanese and other non-English content. The controller sends a deterministic language hint with each query so live and local answers can match the current user prompt language. Markdown rendering is intentionally small and safe: headings, connector/source section labels, bullets, numbered lines, raw URLs, Markdown links, inline code, and bold spans render as structured Svelte nodes rather than raw HTML. Connector labels such as Jira, Slack, GitHub, Google Drive, Notion, SharePoint, and Filesystem render as subtle section rows so long source answers read as grouped report sections without changing the chat layout.

The message list follows new messages only while the user is already near the bottom of the thread. If the user scrolls upward during a long Codex stream, the pane preserves their viewport instead of jumping to the newest progress line.

## Maintenance Notes

- Keep message rendering stable for repeated chat interactions.
- Avoid putting connector ingest logic in these components.
- Update `apps/frontend/src/lib/README.md` if exported chat types or helper contracts change.
