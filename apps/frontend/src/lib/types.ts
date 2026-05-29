export type ServiceStatus = "checking" | "ok" | "unreachable";

export type IngestProvider = "token" | "codex";

export type ConnectorKind = "github" | "slack" | "jira" | "filesystem";

export type CodexConnectorKind = Extract<
  ConnectorKind,
  "github" | "slack" | "jira"
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

export interface IngestEvent {
  id: string;
  type: string;
  source: string;
  source_id: string;
  subject: string;
  occurred_at: string;
}

export interface IngestResult {
  connector: string;
  capabilities: string[];
  event: IngestEvent;
  events?: IngestEvent[];
  event_count?: number;
  preview: string;
  previews?: string[];
  metadata: Record<string, string>;
  metadata_items?: Record<string, string>[];
}

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
