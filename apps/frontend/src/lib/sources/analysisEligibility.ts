import type {
  AnswerSection,
  Artifact,
  ChatQueryResult,
  ConnectorKind,
  ConnectorKnowledge,
} from "$lib/types";
import type { EvidenceBasketItem } from "$lib/workflow/types";

const liveConnectors = new Set<ConnectorKind>([
  "github",
  "jira",
  "slack",
  "notion",
  "sharepoint",
  "googledrive",
]);

const allConnectors = new Set<ConnectorKind>([
  "github",
  "jira",
  "slack",
  "notion",
  "sharepoint",
  "googledrive",
  "filesystem",
]);

export const maxDerivedAnalysisSources = 12;

export interface SkippedAnalysisSource {
  connector: ConnectorKind;
  uri: string;
  reason: string;
}

export interface AnalysisSourcesInput {
  readySources?: ConnectorKnowledge[];
  lastChatResult?: ChatQueryResult | null;
  recentArtifacts?: Artifact[];
  basketItems?: EvidenceBasketItem[];
  derivedLimit?: number;
}

export interface AnalysisSourcesResult {
  eligible: ConnectorKnowledge[];
  skipped: SkippedAnalysisSource[];
  derived: ConnectorKnowledge[];
  basket: ConnectorKnowledge[];
  available: ConnectorKnowledge[];
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

export function buildAnalysisSources({
  readySources = [],
  lastChatResult = null,
  recentArtifacts = [],
  basketItems = [],
  derivedLimit = maxDerivedAnalysisSources,
}: AnalysisSourcesInput): AnalysisSourcesResult {
  const { eligible, skipped } = splitAnalysisSources(readySources);
  const builder = new AnalysisSourceBuilder();

  for (const source of eligible) {
    builder.addExisting(source);
  }

  const addDerived = (source: ConnectorKnowledge | null) => {
    if (builder.derivedCount >= derivedLimit) return;
    builder.addDerived(source);
  };

  if (lastChatResult) {
    for (const section of lastChatResult.answer_sections ?? []) {
      addDerived(sourceFromAnswerSection(section, lastChatResult));
    }

    for (const artifact of lastChatResult.artifacts ?? []) {
      addDerived(sourceFromArtifact(artifact));
    }
  }

  for (const artifact of recentArtifacts) {
    addDerived(sourceFromArtifact(artifact));
  }

  const available = builder.sources();
  const basketBuilder = new AnalysisSourceBuilder();
  for (const item of basketItems) {
    basketBuilder.addDerived(sourceFromBasketItem(item));
  }
  const basket = basketBuilder.sources();
  if (basket.length > 0) {
    return {
      eligible: basket,
      skipped,
      derived: builder.derivedSources(),
      basket,
      available,
    };
  }

  return {
    eligible: available,
    skipped,
    derived: builder.derivedSources(),
    basket: [],
    available,
  };
}

export function analysisSourceCountLabel(count: number) {
  return `${count} concrete source${count === 1 ? "" : "s"}`;
}

class AnalysisSourceBuilder {
  private readonly items: ConnectorKnowledge[] = [];
  private readonly derived: ConnectorKnowledge[] = [];
  private readonly seen = new Set<string>();

  get derivedCount() {
    return this.derived.length;
  }

  addExisting(source: ConnectorKnowledge) {
    const normalized = normalizeAnalysisSource(source);
    if (!normalized || !isAnalysisEligibleSource(normalized)) return;
    this.add(normalized, false);
  }

  addDerived(source: ConnectorKnowledge | null) {
    const normalized = normalizeAnalysisSource(source);
    if (!normalized || !isAnalysisEligibleSource(normalized)) return;
    this.add(normalized, true);
  }

  sources() {
    return this.items.slice();
  }

  derivedSources() {
    return this.derived.slice();
  }

  private add(source: ConnectorKnowledge, derived: boolean) {
    const key = `${source.connector}\u0000${source.uri}`;
    if (this.seen.has(key)) return;
    this.seen.add(key);
    this.items.push(source);
    if (derived) this.derived.push(source);
  }
}

function sourceFromAnswerSection(
  section: AnswerSection,
  result: ChatQueryResult,
): ConnectorKnowledge | null {
  for (const sourceURI of [
    section.source_uri,
    concreteSectionLink(section),
    result.source_uri,
  ]) {
    const cleanURI = trimSourceURI(sourceURI ?? "");
    const connector = firstConnector(
      connectorFromURI(cleanURI),
      section.connector,
      connectorFromSectionLinks(section),
      result.connector,
    );
    const source = makeReadySource(connector, cleanURI);
    if (source && isAnalysisEligibleSource(source)) return source;
  }
  return null;
}

function sourceFromArtifact(artifact: Artifact): ConnectorKnowledge | null {
  for (const sourceURI of [
    artifact.source_uri,
    artifact.metadata?.source_uri,
    artifact.metadata?.source_url,
    artifact.metadata?.url,
  ]) {
    const cleanURI = trimSourceURI(sourceURI ?? "");
    const connector = firstConnector(
      artifact.connector,
      artifact.metadata?.connector,
      connectorFromURI(cleanURI),
    );
    const source = makeReadySource(connector, cleanURI);
    if (source && isAnalysisEligibleSource(source)) return source;
  }
  return null;
}

function concreteSectionLink(section: AnswerSection) {
  for (const link of section.links ?? []) {
    const clean = trimSourceURI(link);
    if (clean && connectorFromURI(clean)) return clean;
  }
  for (const link of section.links ?? []) {
    const clean = trimSourceURI(link);
    if (clean) return clean;
  }
  return "";
}

function connectorFromSectionLinks(section: AnswerSection) {
  for (const link of section.links ?? []) {
    const connector = connectorFromURI(link);
    if (connector) return connector;
  }
  return "";
}

function makeReadySource(
  connector: ConnectorKind | "",
  uri: string,
): ConnectorKnowledge | null {
  if (!connector || !uri) return null;
  return {
    connector,
    uri,
    status: "ready",
  };
}

function sourceFromBasketItem(item: EvidenceBasketItem): ConnectorKnowledge | null {
  const connector = normalizeConnector(item.connector);
  const uri = trimSourceURI(item.uri);
  return makeReadySource(connector, uri);
}

function normalizeAnalysisSource(
  source: ConnectorKnowledge | null,
): ConnectorKnowledge | null {
  if (!source) return null;
  const connector = normalizeConnector(source.connector);
  const uri = trimSourceURI(source.uri);
  if (!connector || !uri) return null;
  return {
    ...source,
    connector,
    uri,
    status: "ready",
  };
}

function normalizeConnector(value?: string): ConnectorKind | "" {
  const clean = (value ?? "").trim().toLowerCase().replace(/[\s_-]+/g, "");
  if (clean === "google" || clean === "gdrive" || clean === "drive") {
    return "googledrive";
  }
  if (clean === "googledrive") return "googledrive";
  if (clean === "github") return "github";
  if (clean === "jira") return "jira";
  if (clean === "slack") return "slack";
  if (clean === "notion") return "notion";
  if (clean === "sharepoint" || clean === "onedrive") return "sharepoint";
  if (clean === "filesystem" || clean === "file") return "filesystem";
  return allConnectors.has(clean as ConnectorKind) ? (clean as ConnectorKind) : "";
}

function firstConnector(...values: Array<string | undefined>) {
  for (const value of values) {
    const connector = normalizeConnector(value);
    if (connector) return connector;
  }
  return "";
}

function connectorFromURI(value?: string): ConnectorKind | "" {
  const clean = trimSourceURI(value ?? "");
  if (!clean) return "";
  const lower = clean.toLowerCase();
  if (/^[a-z][a-z0-9]+-\d+$/i.test(clean)) return "jira";
  if (/^#[a-z0-9_.-]+$/i.test(clean) || lower.startsWith("slack://")) {
    return "slack";
  }
  try {
    const parsed = new URL(clean);
    const host = parsed.host.toLowerCase();
    const path = parsed.pathname.toLowerCase();
    if (host.includes("docs.google.com") || host.includes("drive.google.com")) {
      return "googledrive";
    }
    if (host.includes("atlassian.net") || path.includes("/browse/")) {
      return "jira";
    }
    if (host.includes("slack.com")) return "slack";
    if (host.includes("github.com") || host.includes("api.github.com")) {
      return "github";
    }
    if (host.includes("notion.so") || host.includes("notion.site")) {
      return "notion";
    }
    if (host.includes("sharepoint.com") || host.includes("onedrive.live.com")) {
      return "sharepoint";
    }
  } catch {
    return "";
  }
  return "";
}

function trimSourceURI(value: string) {
  return value.trim().replace(/[.,;:!?"'`)\]}>]+$/g, "");
}
