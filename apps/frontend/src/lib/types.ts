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

// ---- Findings / presentation types ----

export interface FindingsRequest {
  connector: ConnectorKind;
  uri?: string;
  token?: string;
  provider?: IngestProvider;
  role?: string;
  cursor?: string;
  content?: string;
  metadata?: Record<string, string>;
  include_execution?: boolean;
}

export interface FindingsMismatch {
  id?: string;
  entity_name?: string;
  mismatch_type?: string;
  severity?: string;
  description?: string;
  evidence?: string[];
  confidence?: number;
  impact?: string;
  recommended_action?: string;
}

export interface FindingsResult {
  connector?: string;
  uri?: string;
  role?: string;
  summary?: string;
  mismatches?: FindingsMismatch[];
  entity_count?: number;
  mismatch_count?: number;
  trace_id?: string;
  error?: string;
}

// ---- Project / knowledge state types ----

export type KnowledgeStatus = "idle" | "configuring" | "ingesting" | "ready" | "error";

export interface ConnectorKnowledge {
  connector: ConnectorKind;
  uri: string;
  status: KnowledgeStatus;
  lastIngestedAt?: string;
  eventCount?: number;
  error?: string;
}

export interface ProjectState {
  workspacePath: string;
  name: string;
  createdAt: string;
  connectors: ConnectorKnowledge[];
  knowledgeInstalledAt?: string;
}

// ---- Chat types ----

export type ChatRole = "user" | "assistant" | "system";
export type ChatCardKind = "ingest" | "findings" | "status" | "onboarding";

export interface ChatCard {
  kind: ChatCardKind;
  ingestResult?: IngestResult;
  findingsResult?: FindingsResult;
  statusMap?: Record<string, boolean>;
  onboardingConnectors?: ConnectorKnowledge[];
}

export interface ChatMessage {
  id: string;
  role: ChatRole;
  text: string;
  createdAt: string;
  card?: ChatCard;
  loading?: boolean;
}
