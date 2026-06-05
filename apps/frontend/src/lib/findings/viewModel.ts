import type {
  Artifact,
  ChatMessage,
  FindingsMismatch,
  FindingsResult,
} from "$lib/types";

export type ActivityTimeFilter = "24h" | "7d" | "30d" | "all";

export type ActivitySourceGroup = {
  key: string;
  label: string;
  artifacts: Artifact[];
};

export type ActivityEventSummary = {
  preview: string;
  detailText: string;
  facts: string[];
  links: string[];
  rawText: string;
};

export type MessageLine = {
  kind: "heading" | "section" | "number" | "bullet" | "body" | "blank";
  text: string;
};

export type LatestFindingsRun = {
  result: FindingsResult;
  analyzedAt: string;
};

export type FindingTopic =
  | "contract_drift"
  | "requirement_gap"
  | "keyword_signal"
  | "dependency_review"
  | "other";

export type FindingTopicGroup = {
  topic: FindingTopic;
  label: string;
  findings: FindingsMismatch[];
};

export const findingTopicOrder: FindingTopic[] = [
  "contract_drift",
  "requirement_gap",
  "keyword_signal",
  "dependency_review",
  "other",
];

export const findingTopicLabels: Record<FindingTopic, string> = {
  contract_drift: "Contract drift",
  requirement_gap: "Requirement gaps",
  keyword_signal: "Keyword signals",
  dependency_review: "Dependency review",
  other: "Other",
};

export function formatTime(value?: string) {
  if (!value) return "never";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(new Date(value));
}

export function findingDetectedTime(lastAnalysisAt: string) {
  return formatTime(lastAnalysisAt || new Date().toISOString());
}

export function findingEvidenceTime(
  recentArtifacts: Artifact[],
  lastAnalysisAt: string,
) {
  const latest = recentArtifacts
    .map((artifact) => artifact.ingested_at)
    .filter(Boolean)
    .sort()
    .at(-1);
  return formatTime(latest || lastAnalysisAt || new Date().toISOString());
}

export function severityLabel(value?: string) {
  const normalized = (value ?? "review").toLowerCase();
  if (normalized === "high") return "HIGH";
  if (normalized === "medium") return "MEDIUM";
  if (normalized === "low") return "LOW";
  return "REVIEW";
}

export function findingSummary(mismatch: FindingsMismatch | unknown) {
  const record = mismatch as Record<string, unknown>;
  return readableFindingSummary(record);
}

export function findingDescription(mismatch: FindingsMismatch | unknown) {
  const record = mismatch as Record<string, unknown>;
  return String(
    record.description ??
      record.recommended_action ??
      "Review this item against source evidence.",
  );
}

export function findingRecommendedAction(
  mismatch: FindingsMismatch | unknown,
) {
  const record = mismatch as Record<string, unknown>;
  return String(record.recommended_action ?? "");
}

export function findingImpact(mismatch: FindingsMismatch | unknown) {
  const record = mismatch as Record<string, unknown>;
  return String(record.impact ?? "");
}

export function isReviewCandidate(mismatch: FindingsMismatch | unknown) {
  const record = mismatch as Record<string, unknown>;
  const type = String(record.type ?? record.mismatch_type ?? "").toLowerCase();
  const id = String(record.id ?? "").toLowerCase();
  return type === "dependency_review" ||
    type === "dependency_risk" ||
    id.startsWith("dependency_risk:");
}

export function actionableFindings(result: FindingsResult | null | undefined) {
  if (!result) return [];
  return (result.mismatches ?? []).filter((mismatch) => !isReviewCandidate(mismatch));
}

export function reviewCandidates(result: FindingsResult | null | undefined) {
  if (!result) return [];
  const explicit = result.review_candidates ?? [];
  const historical = (result.mismatches ?? []).filter(isReviewCandidate);
  return uniqueByFindingID([...explicit, ...historical]);
}

export function actionableFindingCount(result: FindingsResult | null | undefined) {
  return actionableFindings(result).length;
}

export function reviewCandidateCount(result: FindingsResult | null | undefined) {
  if (!result) return 0;
  return Math.max(result.review_candidate_count ?? 0, reviewCandidates(result).length);
}

export function findingTopic(mismatch: FindingsMismatch | unknown): FindingTopic {
  const record = mismatch as Record<string, unknown>;
  const type = String(record.type ?? record.mismatch_type ?? "").toLowerCase();
  if (type === "cross_layer_contract_drift" || type === "contract_drift") {
    return "contract_drift";
  }
  if (type === "requirement_gap") return "requirement_gap";
  if (type === "keyword_signal") return "keyword_signal";
  if (isReviewCandidate(record)) return "dependency_review";
  return "other";
}

export function groupFindingsByTopic(findings: FindingsMismatch[]) {
  const groups = new Map<FindingTopic, FindingsMismatch[]>();
  for (const topic of findingTopicOrder) {
    groups.set(topic, []);
  }
  for (const finding of findings) {
    groups.get(findingTopic(finding))?.push(finding);
  }
  return findingTopicOrder
    .map((topic) => ({
      topic,
      label: findingTopicLabels[topic],
      findings: groups.get(topic) ?? [],
    }))
    .filter((group) => group.findings.length > 0);
}

export function topActionableFindings(
  result: FindingsResult | null | undefined,
  limit = 3,
) {
  return actionableFindings(result).slice(0, limit);
}

function readableFindingSummary(record: Record<string, unknown>) {
  const summary = String(record.summary ?? record.mismatch_type ?? record.id ?? "Finding");
  if (!summary.includes("event:")) {
    return summary;
  }
  const evidenceNames = evidenceAnchorNames(record.evidence);
  const dependencyMatch = summary.match(/^Service\s+(.+?)\s+depends on\s+(.+?);(.*)$/);
  if (dependencyMatch) {
    const service = evidenceNames[0] ?? compactGraphID(dependencyMatch[1]);
    const dependency = evidenceNames[1] ?? compactGraphID(dependencyMatch[2]);
    return `Service ${service} depends on ${dependency};${dependencyMatch[3]}`;
  }
  return summary.replace(/\bevent:[^\s;]+/g, (value) => compactGraphID(value));
}

function evidenceAnchorNames(value: unknown) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => String(item ?? ""))
    .map((item) => item.split("#").at(-1)?.trim() ?? "")
    .filter(Boolean);
}

function compactGraphID(value: string) {
  const parts = value.trim().split(":").filter(Boolean);
  return parts.at(-1) ?? value;
}

function uniqueByFindingID(findings: FindingsMismatch[]) {
  const seen = new Set<string>();
  const out: FindingsMismatch[] = [];
  for (const finding of findings) {
    const id = String(finding.id ?? findingSummary(finding));
    if (seen.has(id)) continue;
    seen.add(id);
    out.push(finding);
  }
  return out;
}

export function latestFindingsRunFromMessages(
  messages: ChatMessage[],
): LatestFindingsRun | null {
  for (const message of [...messages].reverse()) {
    if (message.card?.kind !== "findings" || !message.card.findingsResult) {
      continue;
    }
    return {
      result: message.card.findingsResult,
      analyzedAt: message.createdAt || new Date().toISOString(),
    };
  }
  return null;
}

export function messageLines(text: string): MessageLine[] {
  return text.split("\n").map((line) => {
    const trimmed = line.replace(/\r/g, "").trim();
    if (trimmed === "") return { kind: "blank", text: "" };
    if (/^\*\*[^*]+\*\*$/.test(trimmed)) {
      return { kind: "heading", text: cleanMarkdown(trimmed) };
    }
    if (/^\d+\.\s+/.test(trimmed)) {
      return { kind: "number", text: trimmed };
    }
    if (/^[-*]\s+/.test(trimmed)) {
      const bulletText = trimmed.replace(/^[-*]\s+/, "");
      if (isSourceSectionLabel(bulletText)) {
        return {
          kind: "section",
          text: bulletText,
        };
      }
      return {
        kind: "bullet",
        text: bulletText,
      };
    }
    return { kind: "body", text: trimmed };
  });
}

function isSourceSectionLabel(value: string) {
  return /^(jira|slack|github|google drive|googledrive|notion|sharepoint|filesystem)$/i.test(
    cleanMarkdown(value).trim(),
  );
}

export function cleanMarkdown(value: string) {
  return value.replace(/\*\*/g, "").replace(/`/g, "");
}

export function previewText(value?: string, max = 360) {
  const text = cleanMarkdown((value ?? "").replace(/\s+/g, " ").trim());
  if (text.length <= max) return text;
  return `${text.slice(0, max).trim()}...`;
}

export function previewMarkdownText(value?: string, max = 360) {
  const text = (value ?? "").replace(/\r/g, "").trim();
  if (text.length <= max) return text;
  return `${text.slice(0, max).trim()}...`;
}

export function markdownBulletList(items: string[] = []) {
  return items
    .flatMap((item) => item.replace(/\r/g, "").split("\n"))
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => (/^([-*]\s+|\d+\.\s+)/.test(item) ? item : `- ${item}`))
    .join("\n");
}

export function artifactOrigin(artifact: Artifact) {
  return artifact.connector === "filesystem" ? "LOCAL" : "SOURCE";
}

export function artifactProvider(artifact: Artifact) {
  return artifact.connector === "filesystem" ? "Local file" : "Codex source";
}

export function artifactSourceLabel(artifact: Artifact) {
  const metadata = artifact.metadata ?? {};
  if (metadata.source_label) {
    return metadata.source_label;
  }
  if (artifact.connector === "slack") {
    return slackSourceLabel(artifact, metadata);
  }
  if (artifact.connector === "googledrive") {
    return googleDriveSourceLabel(artifact, metadata);
  }
  if (artifact.connector === "github") {
    const owner = metadata.github_owner;
    const repo = metadata.github_repo;
    return owner && repo
      ? `${owner}/${repo}`
      : artifact.source_uri || artifact.title || "GitHub";
  }
  return artifact.source_uri || artifact.title || artifact.connector;
}

function slackSourceLabel(artifact: Artifact, metadata: Record<string, string>) {
  const channelName =
    metadata.slack_channel_name ||
    metadata.channel_name ||
    metadata.conversation_name;
  if (channelName) {
    return channelName.startsWith("#") ? channelName : `#${channelName}`;
  }
  const conversation = slackConversationFromText(
    artifact.body || artifact.preview || artifact.title,
  );
  if (conversation) return conversation;
  const channelID =
    metadata.slack_channel_id ||
    metadata.channel_id ||
    slackChannelFromURL(artifact.source_uri);
  if (channelID) return channelID.startsWith("#") ? channelID : `Slack ${channelID}`;
  return readableSourceFallback(artifact, "Slack");
}

function googleDriveSourceLabel(
  artifact: Artifact,
  metadata: Record<string, string>,
) {
  const label =
    metadata.drive_file_name ||
    metadata.google_drive_file_name ||
    metadata.file_name ||
    metadata.filename ||
    metadata.name;
  if (label) return label;
  const fromTitle = readableTitle(artifact.title);
  if (fromTitle) return fromTitle;
  const fromBody = fileNameFromText(artifact.body || artifact.preview);
  if (fromBody) return fromBody;
  return readableSourceFallback(artifact, "Google Drive");
}

function readableSourceFallback(artifact: Artifact, fallback: string) {
  return readableTitle(artifact.title) ||
    fileNameFromText(artifact.source_uri) ||
    artifact.source_uri ||
    fallback;
}

function readableTitle(value?: string) {
  const clean = (value ?? "").trim();
  if (!clean || /^https?:\/\//i.test(clean)) return "";
  return clean;
}

function fileNameFromText(value?: string) {
  const clean = (value ?? "").trim();
  if (!clean) return "";
  const pathMatch = clean.match(
    /([^/\s?#]+?\.(?:pdf|pptx?|xlsx?|csv|docx?|html?|md|txt|json|ya?ml))(?:[?#][^\s]*)?/i,
  );
  return pathMatch?.[1]?.trim() ?? "";
}

function slackConversationFromText(value?: string) {
  const clean = (value ?? "").trim();
  const conversation = clean.match(
    /\bConversation:\s*([^,\n.]+)(?:,\s*([A-Z0-9]+))?/i,
  );
  if (!conversation) return "";
  const name = conversation[1]?.trim();
  const id = conversation[2]?.trim();
  if (name && name.toLowerCase() !== "dm") {
    return name.startsWith("#") ? name : `#${name}`;
  }
  return id ? `Slack DM ${id}` : "Slack DM";
}

function slackChannelFromURL(value?: string) {
  try {
    const url = new URL(value ?? "");
    const match = url.pathname.match(/\/archives\/([^/]+)/);
    return match?.[1] ?? "";
  } catch {
    return "";
  }
}

export function artifactLink(artifact: Artifact) {
  const fields = [
    artifact.source_uri,
    artifact.metadata?.source_uri,
    artifact.metadata?.source_url,
    artifact.metadata?.url,
    artifact.body,
    artifact.preview,
  ];
  for (const field of fields) {
    const match = field?.match(/https?:\/\/[^\s)]+/);
    if (match) return match[0].replace(/[.,;]+$/, "");
  }
  return "";
}

export function activityEventSummary(artifact: Artifact): ActivityEventSummary {
  const rawText = artifact.body || artifact.preview || "";
  const summaryText = artifact.preview || artifact.title || rawText;
  const preview = previewText(summaryText, 720);
  const detailText = previewMarkdownText(summaryText, 1200);
  const links = uniqueValues(extractLinks(rawText || artifact.preview || artifact.source_uri));
  const facts = extractFactLines(rawText)
    .filter((line) => !links.some((link) => line.includes(link)))
    .slice(0, 6);
  return { preview, detailText, facts, links, rawText };
}

export function activityFilterLabel(filter: ActivityTimeFilter) {
  switch (filter) {
    case "24h":
      return "Last 24h";
    case "7d":
      return "Last 7d";
    case "30d":
      return "Last 30d";
    default:
      return "All time";
  }
}

export function normalizeActivityTimeFilter(
  value: string | null | undefined,
): ActivityTimeFilter {
  if (value === "24h" || value === "7d" || value === "30d" || value === "all") {
    return value;
  }
  return "7d";
}

export function filterArtifactsByTime(
  artifacts: Artifact[],
  filter: ActivityTimeFilter,
  now = new Date(),
) {
  if (filter === "all") return [...artifacts];
  const windowMs = activityFilterWindowMs(filter);
  const cutoff = now.getTime() - windowMs;
  return artifacts.filter((artifact) => {
    const time = Date.parse(artifact.ingested_at);
    return Number.isFinite(time) && time >= cutoff;
  });
}

export function groupArtifactsBySource(
  artifacts: Artifact[],
): ActivitySourceGroup[] {
  const groups = new Map<string, ActivitySourceGroup>();
  for (const artifact of artifacts) {
    const label = artifactSourceLabel(artifact);
    const key = `${artifact.connector}:${label}`;
    const group = groups.get(key) ?? {
      key,
      label,
      artifacts: [],
    };
    group.artifacts.push(artifact);
    groups.set(key, group);
  }
  return [...groups.values()].sort((left, right) => {
    const leftTime = latestArtifactTime(left.artifacts);
    const rightTime = latestArtifactTime(right.artifacts);
    return rightTime - leftTime || left.label.localeCompare(right.label);
  });
}

export function artifactDetailRows(artifact: Artifact) {
  const rows = [
    ["Connector", artifact.connector],
    ["Event", artifact.event_type],
    ["Source", artifact.source_uri],
    ["Ingested", formatTime(artifact.ingested_at)],
    ["Schema", artifact.schema_version],
    ["Hash", artifact.content_hash],
  ];
  const metadataRows = Object.entries(artifact.metadata ?? {})
    .filter(([, value]) => value)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => [`metadata.${key}`, value]);
  return [...rows, ...metadataRows].filter(([, value]) => value);
}

function activityFilterWindowMs(filter: ActivityTimeFilter) {
  switch (filter) {
    case "24h":
      return 24 * 60 * 60 * 1000;
    case "30d":
      return 30 * 24 * 60 * 60 * 1000;
    default:
      return 7 * 24 * 60 * 60 * 1000;
  }
}

function latestArtifactTime(artifacts: Artifact[]) {
  return artifacts.reduce((latest, artifact) => {
    const time = Date.parse(artifact.ingested_at);
    return Number.isFinite(time) && time > latest ? time : latest;
  }, 0);
}

function extractLinks(value: string) {
  return [...value.matchAll(/https?:\/\/[^\s)]+/g)].map((match) =>
    match[0].replace(/[.,;]+$/, ""),
  );
}

function extractFactLines(value: string) {
  return value
    .split("\n")
    .map((line) => line.replace(/\r/g, "").trim())
    .map((line) => line.replace(/^[-*]\s+/, "").trim())
    .filter((line) => line.length > 0)
    .filter((line) => !/^source:|^message link:|^channel link:/i.test(line))
    .filter((line) => line.length <= 220)
    .slice(0, 12);
}

function uniqueValues(values: string[]) {
  return [...new Set(values.filter(Boolean))];
}
