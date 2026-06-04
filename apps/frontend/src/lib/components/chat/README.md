# chat components

Svelte components for the main local chat thread.

## Components

| File | Purpose |
| --- | --- |
| `ChatPanel.svelte` | Renders the homepage report-agent pane, compact stream progress, safe Markdown text, structured source cards, source traces, message evidence previews, findings previews, and composer. |
| `InlineText.svelte` | Renders inline links, raw URLs, code, and bold text without using raw HTML injection. |
| `ChatInput.svelte` | Captures and submits the user's chat message. |
| `ChatMessage.svelte` | Renders one chat message and its metadata. |
| `ChatThread.svelte` | Renders the message list and scrolls to the latest message after updates. |

## Usage

These components are presentation-only. API calls, workspace state, and message orchestration should stay in the route or shared frontend helpers such as `$lib/chat/controller`. Query answer metadata renders `ChatQueryResult.provider` so users can distinguish live Codex lookup from local DB evidence.

Pending query progress lives in `ChatMessage.stream` instead of the answer text. `ChatPanel.svelte` renders it as a collapsed stream block by default: running and completed streams show status, the latest Codex-style `›` or `•` progress line, and the final summary when available. Expanding the block shows the full stream transcript in order.

The composer uses a wrapping textarea so long prompts move onto new lines instead of stretching horizontally. Enter inserts a line break, the send button submits, and Ctrl/Cmd+Enter submits from the keyboard.

Query answers always render the plain answer first. When `chatResult.answer_sections` is present, `ChatPanel.svelte` renders one source card per section with connector/source label, source link, summary, facts, open items, coding notes, and links. The panel then shows a lightweight Local DB save status when the backend reports `evidence_save_status`, followed by a source trace with provider, connector, source URI, non-duplicative stream summary, and artifact count. Artifact evidence details remain available when `chatResult.artifacts.length > 0`. Live Codex answers without artifacts are labeled as a live source trace; a separate `Local DB: saved ...; graph updated`, `skipped ...`, or `save failed ...` line tells the user whether the live answer evidence was persisted and whether Graph updated. Findings are not refreshed from chat saves.

Chat message text preserves Japanese and other non-English content. The controller sends a deterministic language hint with each query so live and local answers can match Chinese, Japanese, Korean, or English input. Markdown rendering is intentionally small and safe: headings, connector/source section labels, bullets, numbered lines, raw URLs, Markdown links, inline code, and bold spans render as structured Svelte nodes rather than raw HTML. Connector labels such as Jira, Slack, GitHub, Google Drive, Notion, SharePoint, and Filesystem render as subtle section rows so long source answers read as grouped report sections without changing the chat layout.

The message list follows new messages only while the user is already near the bottom of the thread. If the user scrolls upward during a long Codex stream, the pane preserves their viewport instead of jumping to the newest progress line.

## Maintenance Notes

- Keep message rendering stable for repeated chat interactions.
- Avoid putting connector ingest logic in these components.
- Update `apps/frontend/src/lib/README.md` if exported chat types or helper contracts change.
