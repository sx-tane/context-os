import type { SourceConnectorConfig } from "$lib/types";

export const sourceConnectorConfigs: SourceConnectorConfig[] = [
  {
    connector: "filesystem",
    title: "Filesystem MCP Connector",
    description:
      "Paste one local file or folder path. The connector detects the artifact type and emits stable provenance automatically.",
    defaultUri: "docs/",
    uriLabel: "File or folder path",
    uriPlaceholder: "docs/ or README.md",
    submitLabel: "Ingest file or folder",
    examples: ["docs/", "README.md", "requirements.xlsx", "openapi.yaml"],
    supportedFormats: [
      {
        format: "Folder",
        extensions: "directory path",
        extraction: "Recurses into supported child files",
      },
      {
        format: "Text and Markdown",
        extensions: ".txt, .md",
        extraction: "Read directly",
      },
      {
        format: "Code and config",
        extensions: ".go, .ts, .json, .yaml, .toml, .sql",
        extraction:
          "Read directly; OpenAPI JSON/YAML receives endpoint and schema metadata",
      },
      {
        format: "Spreadsheet",
        extensions: ".xlsx, .csv",
        extraction: "Cell, sheet, row, value, and formula facts",
      },
      {
        format: "Word document",
        extensions: ".docx",
        extraction: "Paragraph text",
      },
      {
        format: "PDF",
        extensions: ".pdf",
        extraction: "Best-effort page text",
      },
      {
        format: "PowerPoint",
        extensions: ".pptx",
        extraction: "Slide text",
      },
    ],
  },
];
