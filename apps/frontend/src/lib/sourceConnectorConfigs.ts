import type { SourceConnectorConfig } from "$lib/types";

export const sourceConnectorConfigs: SourceConnectorConfig[] = [
  {
    connector: "filesystem",
    title: "Filesystem MCP Connector",
    description:
      "Ingest local files or folders, including spreadsheets and OpenAPI specs, with stable path and content-hash provenance.",
    defaultUri: "docs/",
    uriPlaceholder: "/workspace/context-os/docs/",
    examples: ["docs/", "README.md", "requirements.xlsx", "openapi.yaml", "docs/brief.docx"],
    metadataFields: [
      {
        key: "filesystem_include",
        label: "Include pattern",
        placeholder: "docs/**",
      },
      {
        key: "filesystem_exclude",
        label: "Exclude pattern",
        placeholder: "**/node_modules/**",
      },
      {
        key: "filesystem_max_files",
        label: "Max files",
        placeholder: "1000",
      },
      {
        key: "filesystem_max_file_size",
        label: "Max file size bytes",
        placeholder: "10485760",
      },
    ],
    supportedFormats: [
      {
        format: "Folder",
        extensions: "directory path",
        extraction: "Recursive file events with include/exclude and size limits",
      },
      {
        format: "Text and Markdown",
        extensions: ".txt, .md",
        extraction: "Read directly",
      },
      {
        format: "Code and config",
        extensions: ".go, .ts, .json, .yaml, .toml, .sql",
        extraction: "Read directly; OpenAPI JSON/YAML receives endpoint and schema metadata",
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
