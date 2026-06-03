import type { Artifact, FindingsMismatch } from "$lib/types";

export type MessageLine = {
  kind: "heading" | "number" | "bullet" | "body" | "blank";
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
      return {
        kind: "bullet",
        text: trimmed.replace(/^[-*]\s+/, ""),
      };
    }
    return { kind: "body", text: trimmed };
  });
}

export function cleanMarkdown(value: string) {
  return value.replace(/\*\*/g, "").replace(/`/g, "");
}

export function previewText(value?: string, max = 360) {
  const text = cleanMarkdown((value ?? "").replace(/\s+/g, " ").trim());
  if (text.length <= max) return text;
  return `${text.slice(0, max).trim()}...`;
}

export function artifactOrigin(artifact: Artifact) {
  return artifact.connector === "filesystem" ? "LOCAL" : "SOURCE";
}

export function artifactProvider(artifact: Artifact) {
  return artifact.connector === "filesystem" ? "Local file" : "Codex source";
}

export function artifactSourceLabel(artifact: Artifact) {
  const metadata = artifact.metadata ?? {};
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
