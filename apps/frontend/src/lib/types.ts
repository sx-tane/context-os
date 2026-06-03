// Auto-generated API types — run `bun run codegen` to refresh from swagger.json.
import type { definitions } from "$lib/generated/api";

// ---- API types (auto-generated from swagger) ----
export type IngestEvent = definitions["events.Event"];
export type IngestResult = definitions["response.Ingest"];
export type EventType = definitions["events.Type"];

// ---- Frontend-only types ----
export type ServiceStatus = "checking" | "ok" | "unreachable";

export type IngestProvider = "token" | "codex";
export type PresentationRole =
  | "pmo"
  | "presentation_layer"
  | "service_layer"
  | "qa"
  | "architecture";

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

export interface CodexSourceOption {
  id: string;
  label: string;
  uri: string;
  kind: string;
  connector: ConnectorKind;
}

export interface CodexSourceList {
  connector: ConnectorKind;
  provider: "codex";
  sources: CodexSourceOption[];
}

// IngestRequest stays as a unified frontend type covering all connectors.
// The swagger has separate per-connector request types; the frontend collapses them.
export interface IngestRequest {
  workspace_id?: string;
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
  workspace_id?: string;
  connector: ConnectorKind;
  uri?: string;
  token?: string;
  provider?: IngestProvider;
  role?: string;
  cursor?: string;
  content?: string;
  metadata?: Record<string, string>;
  include_execution?: boolean;
  force_refresh?: boolean;
}

export interface FindingsMismatch {
  id?: string;
  type?: string;
  summary?: string;
  entity_name?: string;
  mismatch_type?: string;
  severity?: string;
  description?: string;
  entity_ids?: string[];
  affected_roles?: string[];
  evidence?: string[];
  confidence?: number;
  impact?: string;
  recommended?: string;
  recommended_action?: string;
}

export interface FindingsRoleView {
  role: PresentationRole | string;
  summary: string;
  mismatch_ids: string[];
  next_actions: string[];
  finding_count: number;
}

export interface FindingsPMO {
  facts: string[];
  risks: string[];
  impacts: string[];
  confidence: Record<string, number>;
  evidence: Record<string, string[]>;
  recommended_decisions: string[];
}

export interface FindingsExecution {
  enabled: boolean;
  assistive: boolean;
  summary: string;
  metadata?: Record<string, string>;
  error?: string;
}

export interface FindingsResult {
  connector?: string;
  uri?: string;
  role?: PresentationRole | string;
  trace_id?: string;
  summary?: string;
  mismatches?: FindingsMismatch[];
  event_count?: number;
  mismatch_count?: number;
  severity_count?: Record<"high" | "medium" | "low", number>;
  mismatch_ids?: string[];
  views?: Record<string, FindingsRoleView>;
  pmo?: FindingsPMO;
  execution?: FindingsExecution;
  entity_count?: number;
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

// ---- Workspace API types ----

export interface WorkspaceRecord {
  id: string;
  name: string;
  path: string;
  created_at?: string;
  updated_at?: string;
}

export interface WorkspaceSyncState {
  connector: string;
  source_uri: string;
  cursor?: string;
  last_synced_at?: string;
  event_count?: number;
  status?: string;
}

export interface SourceRegistrationRequest {
  workspace_id: string;
  connector: ConnectorKind;
  source_uri: string;
}

export interface WorkspaceStatus {
  workspace?: WorkspaceRecord;
  workspace_count?: number;
  event_count?: number;
  entity_count?: number;
  relationship_count?: number;
  mismatch_count?: number;
  connector_sync_count?: number;
  audit_count?: number;
  syncs?: WorkspaceSyncState[];
}

export interface WorkspaceList {
  workspaces: WorkspaceRecord[];
  count: number;
}

// ---- Local artifact / chat query types ----

export interface Artifact {
  id: string;
  workspace_id: string;
  connector: string;
  source_uri: string;
  event_type: string;
  title: string;
  body: string;
  preview: string;
  content_hash: string;
  metadata?: Record<string, string>;
  schema_version: string;
  ingested_at: string;
}

export interface ArtifactList {
  workspace_id: string;
  workspace_path: string;
  connector?: string;
  source_uri?: string;
  query?: string;
  count: number;
  artifacts: Artifact[];
}

export interface ChatQueryRequest {
  workspace_id: string;
  workspace_path?: string;
  message: string;
  connector?: string;
  source_uri?: string;
  timezone?: string;
  local_date?: string;
  limit?: number;
}

export interface ChatQueryResult {
  intent: "artifacts" | "findings" | "status" | "unsupported" | string;
  workspace_id: string;
  workspace_path: string;
  connector?: string;
  source_uri?: string;
  provider: "local" | "codex" | string;
  answer: string;
  summary: string;
  range_start?: string;
  range_end?: string;
  artifact_count: number;
  artifacts: Artifact[];
  syncs?: WorkspaceSyncState[];
}

// ---- Graph types ----

export interface GraphEntityCandidate {
  alias: string;
  layer: string;
  confidence: number;
  accepted: boolean;
}

export interface GraphEntity {
  id: string;
  name: string;
  type: string;
  source: string;
  confidence: number;
  needs_human?: boolean;
  conflict_reason?: string;
  evidence?: string[];
  aliases?: string[];
  candidates?: GraphEntityCandidate[];
}

export interface GraphRelationship {
  id: string;
  from_id: string;
  to_id: string;
  kind: string;
  confidence: number;
  evidence?: string[];
  metadata?: Record<string, string>;
}

export interface GraphData {
  workspace_id: string;
  entity_type?: string;
  count: number;
  entity_count?: number;
  relationship_count?: number;
  entities: GraphEntity[];
  relationships?: GraphRelationship[];
}

// ---- Chat types ----

export type ChatRole = "user" | "assistant" | "system";
export type ChatCardKind = "ingest" | "findings" | "status" | "onboarding" | "query";

export interface ChatCard {
  kind: ChatCardKind;
  ingestResult?: IngestResult;
  findingsResult?: FindingsResult;
  chatResult?: ChatQueryResult;
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
