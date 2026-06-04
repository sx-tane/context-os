import type { Artifact, FindingsMismatch } from "$lib/types";

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
  return String(record.summary ?? record.mismatch_type ?? record.id ?? "Finding");
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
    return (
      metadata.slack_channel_id ||
      artifact.source_uri ||
      artifact.title ||
      "Slack"
    );
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
