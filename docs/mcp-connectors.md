# MCP Connector Architecture

All external source integrations in ContextOS are MCP-first connectors. Each connector implements the domain `MCPSourceConnector` contract and converts source-specific input into `document.ingested` events.

## Phase 1 — local-first connectors

These connectors ingest from local paths or authenticated APIs with no hosted infrastructure dependency.

| Connector  | Source                                                                                                 | Capability    | Issue |
| ---------- | ------------------------------------------------------------------------------------------------------ | ------------- | ----- |
| GitHub     | Repository, issues, PRs                                                                                | `repository`  | #7    |
| Slack      | Messages, threads, channels                                                                            | `messages`    | #8    |
| Jira       | Issues, comments, status history                                                                       | `issues`      | #9    |
| OpenAPI    | Endpoint and schema specs                                                                              | `api_spec`    | #10   |
| Excel      | Workbooks, sheets, cells                                                                               | `spreadsheet` | #11   |
| Filesystem | Local files (`.txt`, `.md`, `.go`, `.yaml`, `.json`, `.ts`, `.docx`, `.pdf`, `.pptx`, `.xlsx`, `.csv`) | `files`       | #12   |

## Phase 2 — cloud and knowledge-base connectors

These connectors require OAuth or API token credentials and target cloud-hosted knowledge stores.

| Connector             | Source                                           | Capability | Issue |
| --------------------- | ------------------------------------------------ | ---------- | ----- |
| Google Drive          | Google Docs, Sheets, Slides                      | `files`    | #30   |
| SharePoint / OneDrive | Word, Excel, PowerPoint, PDF via Microsoft Graph | `files`    | #31   |
| Confluence            | Pages and spaces (Cloud and Data Center)         | `docs`     | #32   |
| Notion                | Pages and database entries                       | `docs`     | #33   |

## Connector output

Each connector emits raw ingestion events that are then normalized, classified, extracted, resolved, related, stored in the context graph, and analyzed for delivery mismatches.
