import type { ConnectorKind, ConnectorKnowledge } from "$lib/types";

const liveConnectors = new Set<ConnectorKind>([
  "github",
  "jira",
  "slack",
  "notion",
  "sharepoint",
  "googledrive",
]);

export interface SkippedAnalysisSource {
  connector: ConnectorKind;
  uri: string;
  reason: string;
}

export function sourceSetupURI(
  connector: ConnectorKind,
  manualURI: string,
  enabled: boolean,
) {
  const cleanURI = manualURI.trim();
  if (connector === "filesystem") return cleanURI;
  if (cleanURI) return cleanURI;
  return enabled ? connector : "";
}

export function isBroadConnectorScope(source: Pick<ConnectorKnowledge, "connector" | "uri">) {
  return (
    liveConnectors.has(source.connector) &&
    source.uri.trim().toLowerCase() === source.connector
  );
}

export function isAnalysisEligibleSource(source: ConnectorKnowledge) {
  if (source.status !== "ready") return false;
  if (source.connector === "filesystem") return source.uri.trim() !== "";
  if (!liveConnectors.has(source.connector)) return source.uri.trim() !== "";
  return source.uri.trim() !== "" && !isBroadConnectorScope(source);
}

export function splitAnalysisSources(sources: ConnectorKnowledge[]) {
  const eligible: ConnectorKnowledge[] = [];
  const skipped: SkippedAnalysisSource[] = [];

  for (const source of sources) {
    if (isAnalysisEligibleSource(source)) {
      eligible.push(source);
      continue;
    }
    if (source.status === "ready" && isBroadConnectorScope(source)) {
      skipped.push({
        connector: source.connector,
        uri: source.uri,
        reason: "chat-only live connector scope",
      });
    }
  }

  return { eligible, skipped };
}

export function analysisSourceCountLabel(count: number) {
  return `${count} concrete source${count === 1 ? "" : "s"}`;
}
