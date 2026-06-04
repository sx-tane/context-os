import type {
  Artifact,
  ChatQueryResult,
  ConnectorKnowledge,
  FindingsResult,
  GraphData,
} from "$lib/types";
import {
  buildAnalysisSources,
  type SkippedAnalysisSource,
} from "$lib/sources/analysisEligibility";

export type FindingsInsightState =
  | "not_run"
  | "current"
  | "stale"
  | "no_concrete_sources";

export interface InsightStatusInput {
  readySources?: ConnectorKnowledge[];
  lastChatResult?: ChatQueryResult | null;
  recentArtifacts?: Artifact[];
  graphData?: GraphData | null;
  lastFindings?: FindingsResult | null;
  lastAnalysisAt?: string;
}

export interface InsightStatus {
  readySourceCount: number;
  concreteSourceCount: number;
  derivedConcreteSourceCount: number;
  chatOnlySourceCount: number;
  chatOnlySources: SkippedAnalysisSource[];
  sourceScopeLabel: string;
  activityEventCount: number;
  latestActivityAt: string;
  activityLabel: string;
  activityFreshnessLabel: string;
  graphNodeCount: number;
  graphLinkCount: number;
  hasGraphContext: boolean;
  graphLabel: string;
  graphRefreshLabel: string;
  findingCount: number;
  lastAnalysisAt: string;
  lastAnalysisLabel: string;
  findingsState: FindingsInsightState;
  findingsLabel: string;
  findingsDetailLabel: string;
  findingsMessage: string;
  footerLabel: string;
}

export function buildInsightStatus({
  readySources = [],
  lastChatResult = null,
  recentArtifacts = [],
  graphData = null,
  lastFindings = null,
  lastAnalysisAt = "",
}: InsightStatusInput): InsightStatus {
  const { eligible, skipped, derived } = buildAnalysisSources({
    readySources,
    lastChatResult,
    recentArtifacts,
  });
  const latestActivityAt = latestArtifactTimestamp(recentArtifacts);
  const graphNodeCount = graphData?.entity_count ?? graphData?.count ?? graphData?.entities?.length ?? 0;
  const graphLinkCount = graphData?.relationship_count ?? graphData?.relationships?.length ?? 0;
  const hasGraphContext = graphNodeCount > 0 || graphLinkCount > 0;
  const findingCount = findingsCount(lastFindings);
  const findingsState = deriveFindingsState({
    concreteSourceCount: eligible.length,
    latestActivityAt,
    lastAnalysisAt,
    hasAnalysisResult: lastFindings !== null,
  });
  const sourceScopeLabel = buildSourceScopeLabel(
    eligible.length,
    skipped.length,
    derived.length,
  );
  const activityLabel = buildActivityLabel(recentArtifacts.length);
  const activityFreshnessLabel = latestActivityAt
    ? `latest ${formatInsightTimestamp(latestActivityAt)}`
    : "no saved evidence";
  const graphLabel = `${plural(graphNodeCount, "node")}, ${plural(graphLinkCount, "link")}`;
  const graphRefreshLabel = hasGraphContext
    ? "graph ready"
    : recentArtifacts.length > 0
      ? "waiting for graph evidence"
      : "no graph context";
  const findingsLabel = findingsStateLabel(findingsState);
  const findingsDetailLabel = buildFindingsDetailLabel(
    findingsState,
    findingCount,
    sourceScopeLabel,
  );
  const findingsMessage = buildFindingsMessage(
    findingsState,
    hasGraphContext,
    skipped.length,
  );

  return {
    readySourceCount: readySources.length,
    concreteSourceCount: eligible.length,
    derivedConcreteSourceCount: derived.length,
    chatOnlySourceCount: skipped.length,
    chatOnlySources: skipped,
    sourceScopeLabel,
    activityEventCount: recentArtifacts.length,
    latestActivityAt,
    activityLabel,
    activityFreshnessLabel,
    graphNodeCount,
    graphLinkCount,
    hasGraphContext,
    graphLabel,
    graphRefreshLabel,
    findingCount,
    lastAnalysisAt,
    lastAnalysisLabel: lastAnalysisAt
      ? `last run ${formatInsightTimestamp(lastAnalysisAt)}`
      : "not run",
    findingsState,
    findingsLabel,
    findingsDetailLabel,
    findingsMessage,
    footerLabel: `Activity: ${activityLabel}, ${activityFreshnessLabel} | Graph: ${graphLabel}, ${graphRefreshLabel} | Findings: ${findingsLabel}, ${findingsDetailLabel}`,
  };
}

function deriveFindingsState({
  concreteSourceCount,
  latestActivityAt,
  lastAnalysisAt,
  hasAnalysisResult,
}: {
  concreteSourceCount: number;
  latestActivityAt: string;
  lastAnalysisAt: string;
  hasAnalysisResult: boolean;
}): FindingsInsightState {
  if (concreteSourceCount === 0) return "no_concrete_sources";
  if (!hasAnalysisResult) return "not_run";
  if (latestActivityAt && timestampAfter(latestActivityAt, lastAnalysisAt)) return "stale";
  return "current";
}

function latestArtifactTimestamp(artifacts: Artifact[]) {
  let latest = "";
  let latestMs = Number.NEGATIVE_INFINITY;

  for (const artifact of artifacts) {
    const ms = Date.parse(artifact.ingested_at);
    if (!Number.isFinite(ms) || ms <= latestMs) continue;
    latestMs = ms;
    latest = artifact.ingested_at;
  }

  return latest;
}

function timestampAfter(candidate: string, baseline: string) {
  const candidateMs = Date.parse(candidate);
  const baselineMs = Date.parse(baseline);
  return Number.isFinite(candidateMs) &&
    Number.isFinite(baselineMs) &&
    candidateMs > baselineMs;
}

function findingsCount(findings: FindingsResult | null) {
  if (!findings) return 0;
  return findings.mismatch_count ?? findings.mismatches?.length ?? 0;
}

function buildActivityLabel(count: number) {
  return plural(count, "event");
}

function buildSourceScopeLabel(
  concreteCount: number,
  chatOnlyCount: number,
  derivedConcreteCount: number,
) {
  const sourceLabel = derivedConcreteCount > 0
    ? plural(concreteCount, "concrete evidence source")
    : plural(concreteCount, "concrete source");
  if (chatOnlyCount === 0) return sourceLabel;
  return `${sourceLabel}, ${plural(chatOnlyCount, "chat-only scope")}`;
}

function buildFindingsDetailLabel(
  state: FindingsInsightState,
  findingCount: number,
  sourceScopeLabel: string,
) {
  if (state === "not_run") return sourceScopeLabel;
  if (state === "no_concrete_sources") return sourceScopeLabel;
  return `${plural(findingCount, "finding")}; ${sourceScopeLabel}`;
}

function findingsStateLabel(state: FindingsInsightState) {
  if (state === "current") return "Current";
  if (state === "stale") return "Stale";
  if (state === "no_concrete_sources") return "No concrete sources";
  return "Not run";
}

function buildFindingsMessage(
  state: FindingsInsightState,
  hasGraphContext: boolean,
  chatOnlySourceCount: number,
) {
  if (state === "stale") {
    return "Activity has newer evidence. Run Analysis to refresh findings.";
  }
  if (state === "no_concrete_sources") {
    return chatOnlySourceCount > 0
      ? "Chat-only live connectors can answer chat. Ask about a specific ticket, channel, repo, PR, document, folder, or file so saved evidence becomes analysis-ready."
      : "Connect a concrete source, or ask chat about a specific ticket, channel, repo, PR, document, folder, or file before running Findings.";
  }
  if (state === "not_run") {
    return hasGraphContext
      ? "Graph has context, findings not run yet."
      : "Run Analysis to build findings from concrete sources.";
  }
  return "Findings are current for the latest analyzed evidence.";
}

function formatInsightTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function plural(count: number, singular: string) {
  return `${count} ${singular}${count === 1 ? "" : "s"}`;
}
