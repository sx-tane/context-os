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
  | "googledrive"
  | "notion"
  | "sharepoint";

export type CodexConnectorKind = Extract<
  ConnectorKind,
  "github" | "slack" | "jira" | "googledrive" | "notion" | "sharepoint"
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
  // SharePoint-specific fields (mapped to SharePointIngest struct fields)
  tenant_id?: string;
  client_id?: string;
  client_secret?: string;
}

export interface ApiErrorBody {
  error?: string;
  message?: string;
}

export type PresentationRole =
  | "pmo"
  | "presentation_layer"
  | "service_layer"
  | "qa"
  | "architecture";

export interface RoleSummaryView {
  role: PresentationRole;
  summary: string;
  mismatch_ids: string[];
  next_actions: string[];
  finding_count: number;
}

export interface PMOSummary {
  facts: string[];
  risks: string[];
  impacts: string[];
  confidence: Record<string, number>;
  evidence: Record<string, string[]>;
  recommended_decisions: string[];
}

export interface Mismatch {
  id: string;
  type: string;
  summary: string;
  entity_ids: string[];
  severity: string;
  confidence: number;
  impact: string;
  evidence: string[];
  affected_roles: string[];
  recommended: string;
}

export interface ExecutionEvidence {
  enabled: boolean;
  assistive: boolean;
  summary: string;
  metadata: Record<string, string>;
  error?: string;
}

export interface FindingsViews {
  pmo: RoleSummaryView;
  presentation_layer: RoleSummaryView;
  service_layer: RoleSummaryView;
  qa: RoleSummaryView;
  architecture: RoleSummaryView;
}

export interface FindingsRequest {
  connector: ConnectorKind;
  uri?: string;
  content?: string;
  provider?: IngestProvider;
  token?: string;
  role?: PresentationRole;
  include_execution?: boolean;
  metadata?: Record<string, string>;
}

export interface FindingsResult {
  connector: string;
  uri: string;
  role: PresentationRole;
  trace_id: string;
  summary: string;
  mismatch_count: number;
  severity_count: Record<string, number>;
  mismatch_ids: string[];
  mismatches: Mismatch[];
  views: FindingsViews;
  pmo: PMOSummary;
  execution: ExecutionEvidence;
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
