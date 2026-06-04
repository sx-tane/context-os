import {
  analysisSourceCountLabel,
  buildAnalysisSources,
  isAnalysisEligibleSource,
  isBroadConnectorScope,
} from "$lib/sources/analysisEligibility";
import {
  artifactLink,
  artifactSourceLabel,
  findingRecommendedAction,
  findingSummary,
  formatTime,
} from "$lib/findings/viewModel";
import type {
  AnswerSection,
  Artifact,
  ChatQueryResult,
  CodexPlugin,
  ConnectorKind,
  ConnectorKnowledge,
  FindingsMismatch,
  FindingsResult,
  GraphData,
  WorkspaceStatus,
} from "$lib/types";
import type {
  EvidenceBasketItem,
  FindingActionItem,
} from "$lib/workflow/types";

export interface AnalysisPreviewInput {
  readySources?: ConnectorKnowledge[];
  lastChatResult?: ChatQueryResult | null;
  recentArtifacts?: Artifact[];
  basketItems?: EvidenceBasketItem[];
}

export interface AnalysisPreviewRow {
  id: string;
  connector: ConnectorKind;
  uri: string;
  label: string;
  status: "included" | "available" | "skipped";
  origin: "basket" | "source" | "evidence" | "chat-only";
}

export interface AnalysisPreviewModel {
  included: AnalysisPreviewRow[];
  available: AnalysisPreviewRow[];
  skipped: AnalysisPreviewRow[];
  summary: string;
  hasBasketSelection: boolean;
}

export interface ActivityFilterState {
  connector?: string;
  sourceURI?: string;
  evidenceType?: string;
  keyword?: string;
}

export interface SourceHealthInput {
  readySources?: ConnectorKnowledge[];
  recentArtifacts?: Artifact[];
  workspaceStatus?: WorkspaceStatus | null;
  codexLoggedIn?: boolean;
  codexPlugins?: CodexPlugin[];
}

export interface SourceHealthRow {
  id: string;
  connector: ConnectorKind;
  uri: string;
  label: string;
  status: "analysis-ready" | "has-concrete-evidence" | "broad-chat-only" | "connected" | "needs-attention";
  detail: string;
}

const connectorOrder: ConnectorKind[] = [
  "github",
  "jira",
  "slack",
  "googledrive",
  "notion",
  "sharepoint",
  "filesystem",
];

export function buildAnalysisPreview(input: AnalysisPreviewInput): AnalysisPreviewModel {
  const sources = buildAnalysisSources(input);
  const basketKeys = new Set(sources.basket.map(sourceKey));
  const included = sources.eligible.map((source) =>
    previewRow(source, basketKeys.size ? "basket" : previewOrigin(source, sources.derived), "included"),
  );
  const available = sources.available
    .filter((source) => !basketKeys.has(sourceKey(source)))
    .map((source) => previewRow(source, previewOrigin(source, sources.derived), "available"));
  const skipped = sources.skipped.map((source) => ({
    id: `skipped:${source.connector}:${source.uri}`,
    connector: source.connector,
    uri: source.uri,
    label: `${source.connector}:${source.uri}`,
    status: "skipped" as const,
    origin: "chat-only" as const,
  }));
  const summary = included.length
    ? `${analysisSourceCountLabel(included.length)} ready${basketKeys.size ? " from basket" : ""}`
    : skipped.length
      ? "No concrete analysis sources; chat-only scopes are available for chat"
      : "No analysis sources ready";
  return {
    included,
    available,
    skipped,
    summary,
    hasBasketSelection: basketKeys.size > 0,
  };
}

export function filterActivityArtifacts(
  artifacts: Artifact[],
  filters: ActivityFilterState,
) {
  const connector = (filters.connector ?? "").trim().toLowerCase();
  const sourceURI = (filters.sourceURI ?? "").trim().toLowerCase();
  const evidenceType = (filters.evidenceType ?? "").trim().toLowerCase();
  const keyword = (filters.keyword ?? "").trim().toLowerCase();
  return artifacts.filter((artifact) => {
    if (connector && artifact.connector.toLowerCase() !== connector) return false;
    if (sourceURI && !artifact.source_uri.toLowerCase().includes(sourceURI)) return false;
    if (evidenceType && activityEvidenceType(artifact).toLowerCase() !== evidenceType) return false;
    if (keyword && !artifactSearchText(artifact).includes(keyword)) return false;
    return true;
  });
}

export function activityEvidenceType(artifact: Artifact) {
  return artifact.metadata?.evidence_kind || artifact.event_type || "event";
}

export function buildSourceHealth(input: SourceHealthInput): SourceHealthRow[] {
  const readySources = input.readySources ?? [];
  const recentArtifacts = input.recentArtifacts ?? [];
  const syncs = input.workspaceStatus?.syncs ?? [];
  const rows = new Map<string, SourceHealthRow>();

  for (const source of readySources) {
    const row = sourceHealthFromSource(source, recentArtifacts, input.codexLoggedIn ?? false);
    rows.set(row.id, row);
  }

  for (const sync of syncs) {
    const connector = normalizeConnector(sync.connector);
    if (!connector) continue;
    const source: ConnectorKnowledge = {
      connector,
      uri: sync.source_uri,
      status: sync.status === "error" ? "error" : "ready",
      eventCount: sync.event_count,
      error: sync.last_error,
    };
    const row = sourceHealthFromSource(source, recentArtifacts, input.codexLoggedIn ?? false);
    rows.set(row.id, row);
  }

  for (const artifact of recentArtifacts) {
    const connector = normalizeConnector(artifact.connector);
    if (!connector) continue;
    const source: ConnectorKnowledge = {
      connector,
      uri: artifact.source_uri,
      status: "ready",
    };
    const row = sourceHealthFromSource(source, recentArtifacts, input.codexLoggedIn ?? false);
    rows.set(row.id, row);
  }

  return [...rows.values()].sort(
    (left, right) =>
      connectorOrder.indexOf(left.connector) - connectorOrder.indexOf(right.connector) ||
      left.uri.localeCompare(right.uri),
  );
}

export function basketItemFromAnswerSection(
  section: AnswerSection,
  messageID = "",
  now = new Date(),
): EvidenceBasketItem | null {
  const connector = normalizeConnector(section.connector || connectorFromURI(section.source_uri || firstURL(section.links)));
  const uri = cleanURI(section.source_uri || firstURL(section.links));
  if (!connector || !uri) return null;
  return {
    id: `${connector}:${uri}`,
    connector,
    uri,
    label: section.source_label || uri,
    origin: "chat",
    messageId: messageID || undefined,
    addedAt: now.toISOString(),
  };
}

export function basketItemFromArtifact(
  artifact: Artifact,
  now = new Date(),
): EvidenceBasketItem | null {
  const connector = normalizeConnector(artifact.connector);
  const uri = cleanURI(artifact.source_uri || artifact.metadata?.source_uri || artifact.metadata?.source_url || artifactLink(artifact));
  if (!connector || !uri) return null;
  return {
    id: `${connector}:${uri}`,
    connector,
    uri,
    label: artifactSourceLabel(artifact),
    origin: "activity",
    artifactId: artifact.id,
    addedAt: now.toISOString(),
  };
}

export function mergeBasketItem(
  current: EvidenceBasketItem[],
  item: EvidenceBasketItem,
) {
  return [item, ...current.filter((existing) => existing.id !== item.id)].slice(0, 24);
}

export function askChatPromptForEvidence(connector: string, uri: string, label = "") {
  const source = `${connector}:${uri}`;
  return `Ask about this evidence source ${source}${label ? ` (${label})` : ""}: `;
}

export function findingActionFor(
  actions: FindingActionItem[],
  findingID: string,
): FindingActionItem {
  return actions.find((item) => item.findingId === findingID) ?? {
    findingId: findingID,
    status: "open",
    updatedAt: new Date(0).toISOString(),
  };
}

export function nextFindingActionStatus(status: FindingActionItem["status"]) {
  if (status === "open") return "checking";
  if (status === "checking") return "done";
  return "open";
}

export function findingShareText(
  finding: FindingsMismatch,
  action: FindingActionItem,
) {
  const recommended = findingRecommendedAction(finding);
  return [
    `Finding: ${findingSummary(finding)}`,
    `Status: ${action.status}`,
    recommended ? `Recommended action: ${recommended}` : "",
    finding.evidence?.length ? `Evidence: ${finding.evidence.join(", ")}` : "",
  ].filter(Boolean).join("\n");
}

export function buildWorkspaceSnapshotMarkdown(input: {
  workspacePath: string;
  preview: AnalysisPreviewModel;
  sourceHealth: SourceHealthRow[];
  findings: FindingsResult | null;
  actions: FindingActionItem[];
  graphData: GraphData | null;
  recentArtifacts: Artifact[];
  basketItems: EvidenceBasketItem[];
}) {
  const findings = input.findings?.mismatches ?? [];
  return [
    `# ContextOS Snapshot: ${input.workspacePath}`,
    "",
    `Generated: ${formatTime(new Date().toISOString())}`,
    "",
    "## Analysis",
    `- ${input.preview.summary}`,
    `- Basket items: ${input.basketItems.length}`,
    ...input.preview.included.map((row) => `- Included: ${row.connector}:${row.uri}`),
    "",
    "## Source Health",
    ...input.sourceHealth.map((row) => `- ${row.label}: ${row.status} (${row.detail})`),
    "",
    "## Findings",
    ...(findings.length
      ? findings.map((finding) => {
          const action = findingActionFor(input.actions, String(finding.id ?? findingSummary(finding)));
          return `- [${action.status}] ${findingSummary(finding)}`;
        })
      : ["- No findings loaded"]),
    "",
    "## Graph",
    `- Nodes: ${input.graphData?.entity_count ?? input.graphData?.entities?.length ?? 0}`,
    `- Links: ${input.graphData?.relationship_count ?? input.graphData?.relationships?.length ?? 0}`,
    "",
    "## Recent Activity",
    ...input.recentArtifacts.slice(0, 12).map((artifact) =>
      `- ${artifact.connector}:${artifact.source_uri} — ${artifact.title || artifact.preview || artifact.event_type}`,
    ),
  ].join("\n");
}

function previewRow(
  source: ConnectorKnowledge,
  origin: AnalysisPreviewRow["origin"],
  status: AnalysisPreviewRow["status"],
): AnalysisPreviewRow {
  return {
    id: `${status}:${source.connector}:${source.uri}`,
    connector: source.connector,
    uri: source.uri,
    label: `${source.connector}:${source.uri}`,
    status,
    origin,
  };
}

function previewOrigin(
  source: ConnectorKnowledge,
  derived: ConnectorKnowledge[],
): AnalysisPreviewRow["origin"] {
  return derived.some((item) => sourceKey(item) === sourceKey(source)) ? "evidence" : "source";
}

function sourceHealthFromSource(
  source: ConnectorKnowledge,
  artifacts: Artifact[],
  codexLoggedIn: boolean,
): SourceHealthRow {
  const hasEvidence = artifacts.some((artifact) =>
    artifact.connector === source.connector && artifact.source_uri === source.uri,
  );
  if (source.status === "error") {
    return healthRow(source, "needs-attention", source.error || "Source reports an error");
  }
  if (source.connector !== "filesystem" && !codexLoggedIn) {
    return healthRow(source, "needs-attention", "Codex login or connector reauth needed");
  }
  if (isAnalysisEligibleSource(source)) {
    return healthRow(source, "analysis-ready", hasEvidence ? "Concrete source with saved evidence" : "Concrete source ready for analysis");
  }
  if (isBroadConnectorScope(source)) {
    return healthRow(source, hasEvidence ? "has-concrete-evidence" : "broad-chat-only", hasEvidence ? "Concrete evidence found in Activity" : "Broad connector scope; chat-only until evidence is concrete");
  }
  return healthRow(source, hasEvidence ? "has-concrete-evidence" : "connected", hasEvidence ? "Saved evidence exists" : "Connected source");
}

function healthRow(
  source: ConnectorKnowledge,
  status: SourceHealthRow["status"],
  detail: string,
): SourceHealthRow {
  return {
    id: `${source.connector}:${source.uri}`,
    connector: source.connector,
    uri: source.uri,
    label: `${source.connector}:${source.uri || source.connector}`,
    status,
    detail,
  };
}

function sourceKey(source: Pick<ConnectorKnowledge, "connector" | "uri">) {
  return `${source.connector}:${source.uri}`;
}

function artifactSearchText(artifact: Artifact) {
  return [
    artifact.connector,
    artifact.source_uri,
    artifact.event_type,
    artifact.title,
    artifact.preview,
    artifact.body,
    ...Object.values(artifact.metadata ?? {}),
  ].join(" ").toLowerCase();
}

function normalizeConnector(value?: string): ConnectorKind | "" {
  const clean = (value ?? "").trim().toLowerCase().replace(/[\s_-]+/g, "");
  if (clean === "google" || clean === "gdrive" || clean === "drive") return "googledrive";
  if (clean === "github" || clean === "jira" || clean === "slack" || clean === "filesystem" || clean === "googledrive" || clean === "notion" || clean === "sharepoint") {
    return clean;
  }
  return "";
}

function connectorFromURI(value?: string) {
  const clean = cleanURI(value);
  if (/^[a-z][a-z0-9]+-\d+$/i.test(clean)) return "jira";
  if (/^#[a-z0-9_.-]+$/i.test(clean) || clean.toLowerCase().startsWith("slack://")) return "slack";
  try {
    const url = new URL(clean);
    const host = url.host.toLowerCase();
    if (host.includes("github.com")) return "github";
    if (host.includes("slack.com")) return "slack";
    if (host.includes("atlassian.net")) return "jira";
    if (host.includes("docs.google.com") || host.includes("drive.google.com")) return "googledrive";
    if (host.includes("notion.so") || host.includes("notion.site")) return "notion";
    if (host.includes("sharepoint.com") || host.includes("onedrive.live.com")) return "sharepoint";
  } catch {
    return "";
  }
  return "";
}

function firstURL(values?: string[]) {
  return values?.find((value) => /^https?:\/\//i.test(value)) ?? values?.[0] ?? "";
}

function cleanURI(value?: string) {
  return (value ?? "").trim().replace(/[.,;:!?"'`)\]}>]+$/g, "");
}
