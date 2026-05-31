// Auto-generated API types — run `bun run codegen` to refresh from swagger.json.
import type { definitions } from "$lib/generated/api";

// ---- API types (auto-generated from swagger) ----
export type IngestEvent = definitions["events.Event"];
export type IngestResult = definitions["response.Ingest"];
export type EventType = definitions["events.Type"];

// ---- Frontend-only types ----
export type ServiceStatus = "checking" | "ok" | "unreachable";

export type IngestProvider = "token" | "codex";

export type ConnectorKind =
  | "github"
  | "slack"
  | "jira"
  | "filesystem"
  | "googledrive";

export type CodexConnectorKind = Extract<
  ConnectorKind,
  "github" | "slack" | "jira" | "googledrive"
>;

export type DirectSourceConnectorKind = Exclude<
  ConnectorKind,
  CodexConnectorKind
>;

export interface CodexPlugin {
  name: string;
  installed: boolean;
  enabled: boolean;
}

// IngestRequest stays as a unified frontend type covering all connectors.
// The swagger has separate per-connector request types; the frontend collapses them.
export interface IngestRequest {
  uri: string;
  token?: string;
  provider: IngestProvider;
  content?: string;
  cursor?: string;
  metadata?: Record<string, string>;
}

export interface ApiErrorBody {
  error?: string;
  message?: string;
}

export interface SourceMetadataField {
  key: string;
  label: string;
  placeholder?: string;
  type?: "text" | "password";
  defaultValue?: string;
}

export interface SupportedFormat {
  format: string;
  extensions: string;
  extraction: string;
}

export interface SourceConnectorConfig {
  connector: DirectSourceConnectorKind;
  title: string;
  description: string;
  examples?: string[];
  defaultUri?: string;
  uriLabel?: string;
  uriPlaceholder?: string;
  submitLabel?: string;
  uploadEnabled?: boolean;
  tokenLabel?: string;
  tokenPlaceholder?: string;
  contentLabel?: string;
  contentPlaceholder?: string;
  metadataFields?: SourceMetadataField[];
  supportedFormats?: SupportedFormat[];
}
