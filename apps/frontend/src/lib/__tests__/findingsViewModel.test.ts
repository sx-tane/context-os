import {
  artifactLink,
  artifactOrigin,
  artifactProvider,
  artifactSourceLabel,
  findingDescription,
  findingImpact,
  findingRecommendedAction,
  findingSummary,
  messageLines,
  previewText,
  severityLabel,
} from "../findingsViewModel";

import type { Artifact } from "../types";

describe("severityLabel", () => {
  it("normalizes known severities and falls back to review", () => {
    expect(severityLabel("high")).toBe("HIGH");
    expect(severityLabel("medium")).toBe("MEDIUM");
    expect(severityLabel("low")).toBe("LOW");
    expect(severityLabel("unknown")).toBe("REVIEW");
  });
});

describe("finding display helpers", () => {
  it("returns summary, description, impact, and action with stable fallbacks", () => {
    const finding = {
      id: "m1",
      mismatch_type: "contract_drift",
      recommended_action: "Update the contract",
      impact: "QA may test the wrong field",
    };

    expect(findingSummary(finding)).toBe("contract_drift");
    expect(findingDescription(finding)).toBe("Update the contract");
    expect(findingImpact(finding)).toBe("QA may test the wrong field");
    expect(findingRecommendedAction(finding)).toBe("Update the contract");
    expect(findingSummary({})).toBe("Finding");
  });
});

describe("messageLines", () => {
  it("parses simple markdown-like chat output into display lines", () => {
    expect(messageLines("**Heading**\n1. First\n- Bullet\nBody")).toEqual([
      { kind: "heading", text: "Heading" },
      { kind: "number", text: "1. First" },
      { kind: "bullet", text: "Bullet" },
      { kind: "body", text: "Body" },
    ]);
  });

  it("preserves bilingual and Japanese lines", () => {
    expect(messageLines("English / 中文\n中文")).toEqual([
      { kind: "body", text: "English / 中文" },
      { kind: "body", text: "中文" },
    ]);
  });
});

describe("artifact display helpers", () => {
  it("labels local filesystem artifacts separately from plugin-backed sources", () => {
    const local = makeArtifact({ connector: "filesystem" });
    const remote = makeArtifact({ connector: "github" });

    expect(artifactOrigin(local)).toBe("LOCAL");
    expect(artifactProvider(local)).toBe("Local file");
    expect(artifactOrigin(remote)).toBe("SOURCE");
    expect(artifactProvider(remote)).toBe("Codex source");
  });

  it("builds source labels and source links from artifact metadata", () => {
    const artifact = makeArtifact({
      connector: "github",
      source_uri: "repo#1",
      metadata: {
        github_owner: "context-os",
        github_repo: "app",
        source_url: "https://example.test/pr/1.",
      },
    });

    expect(artifactSourceLabel(artifact)).toBe("context-os/app");
    expect(artifactLink(artifact)).toBe("https://example.test/pr/1");
  });
});

describe("previewText", () => {
  it("cleans markdown and truncates long previews", () => {
    expect(previewText("**Important** `field`", 20)).toBe("Important field");
    expect(previewText("one two three four", 8)).toBe("one two...");
  });
});

function makeArtifact(overrides: Partial<Artifact> = {}): Artifact {
  return {
    id: "a1",
    workspace_id: "workspace",
    connector: "github",
    source_uri: "repo",
    event_type: "document.ingested",
    title: "Artifact",
    body: "Body",
    preview: "Preview",
    content_hash: "hash",
    metadata: {},
    schema_version: "v1",
    ingested_at: "2026-01-01T00:00:00.000Z",
    ...overrides,
  };
}
