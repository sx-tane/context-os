# Frontend App

SvelteKit application surface for ContextOS presentation views.

Current local workflow:

- probes API and worker health from the main page;
- shows Codex CLI install/login/plugin status;
- supports Codex device-auth login and plugin re-auth streams;
- runs GitHub MCP ingestion through either direct token/env auth or the Codex GitHub plugin;
- runs Slack MCP ingestion through direct token/env/OAuth auth or the Codex Slack plugin;
- runs Jira/Rovo MCP ingestion through the Atlassian Rovo Codex plugin by default, with direct API token/env auth still available for local checks;
- runs filesystem MCP ingestion for local files, recursive folders, spreadsheets, OpenAPI specs, and document formats with optional include/exclude rules and folder guardrails;
- renders single-file and folder ingestion events, previews, metadata, and SSE progress logs.

## Filesystem Supported Formats

| Format            | Extensions                                      | Extraction                                                                       |
| ----------------- | ----------------------------------------------- | -------------------------------------------------------------------------------- |
| Folder            | Directory path                                  | Recursive child-file events with stable per-file IDs                             |
| Text and Markdown | `.txt`, `.md`                                   | Read directly                                                                    |
| Code and config   | `.go`, `.ts`, `.json`, `.yaml`, `.toml`, `.sql` | Read directly; OpenAPI JSON/YAML gets endpoint and schema metadata when detected |
| Spreadsheet       | `.xlsx`, `.csv`                                 | Cell, sheet, row, value, and formula facts                                       |
| Word document     | `.docx`                                         | Paragraph text                                                                   |
| PDF               | `.pdf`                                          | Best-effort page text                                                            |
| PowerPoint        | `.pptx`                                         | Slide text                                                                       |

Production responsibility:

- show role-specific PMO, frontend, backend, QA, and architecture views;
- keep findings tied to evidence and impacted artifacts;
- separate facts, confidence, impact, and recommended next actions;
- support fast local workflows for inspecting delivery misalignment.
