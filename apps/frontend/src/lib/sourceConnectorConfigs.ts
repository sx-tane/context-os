import type { SourceConnectorConfig } from "$lib/types";

export const sourceConnectorConfigs: SourceConnectorConfig[] = [
  {
    connector: "filesystem",
    title: "Filesystem MCP Connector",
    description:
      "Choose files or a folder from your computer. Server-visible paths stay available for local developer workflows.",
    defaultUri: "docs/",
    uriLabel: "File or folder path",
    uriPlaceholder: "docs/ or README.md",
    submitLabel: "Ingest server path",
    uploadEnabled: true,
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
