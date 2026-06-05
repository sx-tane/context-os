import {
  activityFilterLabel,
  activityEventSummary,
  artifactLink,
  artifactDetailRows,
  artifactOrigin,
  artifactProvider,
  artifactSourceLabel,
  actionableFindings,
  filterArtifactsByTime,
  findingDescription,
  findingImpact,
  findingRecommendedAction,
  findingSummary,
  findingTopic,
  groupFindingsByTopic,
  groupArtifactsBySource,
  latestFindingsRunFromMessages,
  markdownBulletList,
  messageLines,
  normalizeActivityTimeFilter,
  previewMarkdownText,
  previewText,
  reviewCandidateCount,
  reviewCandidates,
  severityLabel,
  topActionableFindings,
} from "../findings/viewModel";

import type { Artifact, ChatMessage, FindingsResult } from "../types";

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

  it("replaces raw graph IDs in dependency titles with evidence labels", () => {
    const finding = {
      summary:
        "Service event:5afef31ce24f91d7815af82c4e3567d97d7fb620a8ac9ed6fc1cba6889e4aee4:service:wbcveritranscreditapiservice depends on event:5afef31ce24f91d7815af82c4e3567d97d7fb620a8ac9ed6fc1cba6889e4aee4:dependency:orderidrepository; confirm the dependency is healthy and owned",
      evidence: [
        "event:5afef31ce24f91d7815af82c4e3567d97d7fb620a8ac9ed6fc1cba6889e4aee4#WbcVeritransCreditApiService",
        "event:5afef31ce24f91d7815af82c4e3567d97d7fb620a8ac9ed6fc1cba6889e4aee4#OrderIdRepository",
      ],
    };

    expect(findingSummary(finding)).toBe(
      "Service WbcVeritransCreditApiService depends on OrderIdRepository; confirm the dependency is healthy and owned",
    );
  });
});

describe("finding quality helpers", () => {
  it("splits dependency review candidates from actionable findings", () => {
    const result: FindingsResult = {
      mismatches: [
        { id: "m1", type: "requirement_gap", summary: "Missing API edge" },
        { id: "dependency_risk:r1", type: "dependency_risk", summary: "Service depends on Repo" },
      ],
      review_candidates: [
        { id: "dependency_risk:r2", type: "dependency_review", summary: "Service depends on DB" },
      ],
      mismatch_count: 1,
      review_candidate_count: 1,
    };

    expect(actionableFindings(result).map((finding) => finding.id)).toEqual(["m1"]);
    expect(reviewCandidates(result).map((finding) => finding.id)).toEqual([
      "dependency_risk:r2",
      "dependency_risk:r1",
    ]);
    expect(reviewCandidateCount(result)).toBe(2);
    expect(topActionableFindings(result, 3)).toHaveLength(1);
  });

  it("groups findings by topic in the default decision order", () => {
    const groups = groupFindingsByTopic([
      { id: "k1", type: "keyword_signal" },
      { id: "d1", type: "dependency_review" },
      { id: "r1", type: "requirement_gap" },
      { id: "c1", type: "cross_layer_contract_drift" },
      { id: "o1", type: "unknown" },
    ]);

    expect(groups.map((group) => group.topic)).toEqual([
      "contract_drift",
      "requirement_gap",
      "keyword_signal",
      "dependency_review",
      "other",
    ]);
    expect(findingTopic({ type: "dependency_risk" })).toBe("dependency_review");
  });
});

describe("latestFindingsRunFromMessages", () => {
  it("restores the newest findings result from cached chat cards", () => {
    const messages: ChatMessage[] = [
      makeMessage("old", "2026-06-04T09:00:00.000Z", 1),
      {
        id: "plain",
        role: "assistant",
        text: "No card",
        createdAt: "2026-06-04T09:30:00.000Z",
      },
      makeMessage("new", "2026-06-04T10:00:00.000Z", 45),
    ];

    const restored = latestFindingsRunFromMessages(messages);

    expect(restored?.analyzedAt).toBe("2026-06-04T10:00:00.000Z");
    expect(restored?.result.mismatch_count).toBe(45);
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

  it("promotes connector bullets to section lines without changing content bullets", () => {
    expect(messageLines("- Jira\n- Issue: BKGDEV-8096\n- 予約一覧")).toEqual([
      { kind: "section", text: "Jira" },
      { kind: "bullet", text: "Issue: BKGDEV-8096" },
      { kind: "bullet", text: "予約一覧" },
    ]);
  });

  it("preserves raw URLs, inline code, and non-English markdown text", () => {
    expect(messageLines("See `field_id` at https://example.test/docs\n- Google Drive")).toEqual([
      { kind: "body", text: "See `field_id` at https://example.test/docs" },
      { kind: "section", text: "Google Drive" },
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

  it("prefers structured source labels saved on live evidence", () => {
    const artifact = makeArtifact({
      connector: "googledrive",
      source_uri: "https://docs.google.com/spreadsheets/d/abc/edit",
      metadata: {
        source_label: "Google Drive · BKGDEV-8096_帳票項目のマッピング確認.xlsx",
      },
    });

    expect(artifactSourceLabel(artifact)).toBe(
      "Google Drive · BKGDEV-8096_帳票項目のマッピング確認.xlsx",
    );
  });

  it("uses Google Drive file names instead of long document URLs", () => {
    const artifact = makeArtifact({
      connector: "googledrive",
      source_uri: "https://drive.google.com/file/d/abc/view?usp=drivesdk",
      title: "https://drive.google.com/file/d/abc/view?usp=drivesdk",
      body: [
        "The provided URL points to a file, not a folder: transaction_history.html.",
        "I used its parent folder tables as the folder source.",
      ].join(" "),
    });

    expect(artifactSourceLabel(artifact)).toBe("transaction_history.html");
  });

  it("uses Slack channel or conversation names instead of archive URLs", () => {
    const channelArtifact = makeArtifact({
      connector: "slack",
      source_uri: "https://gxp.slack.com/archives/C07TEAMCHAT/p1780386781885339",
      metadata: { channel_name: "payment-flow" },
    });
    const dmArtifact = makeArtifact({
      connector: "slack",
      source_uri: "https://gxp.slack.com/archives/D07E58E08SF/p1780386781885339",
      body: "Read-only Slack context retrieved. Source - Conversation: DM, D07E58E08SF - Linked message: 1780386781.885339",
    });

    expect(artifactSourceLabel(channelArtifact)).toBe("#payment-flow");
    expect(artifactSourceLabel(dmArtifact)).toBe("Slack DM D07E58E08SF");
  });
});

describe("activity display helpers", () => {
  it("normalizes, labels, and applies local time filters", () => {
    const now = new Date("2026-06-03T12:00:00.000Z");
    const artifacts = [
      makeArtifact({ id: "recent", ingested_at: "2026-06-03T10:00:00.000Z" }),
      makeArtifact({ id: "old", ingested_at: "2026-05-01T10:00:00.000Z" }),
    ];

    expect(normalizeActivityTimeFilter("bad")).toBe("7d");
    expect(activityFilterLabel("30d")).toBe("Last 30d");
    expect(filterArtifactsByTime(artifacts, "24h", now).map((item) => item.id)).toEqual([
      "recent",
    ]);
    expect(filterArtifactsByTime(artifacts, "all", now)).toHaveLength(2);
  });

  it("groups activity by source and exposes inspectable detail rows", () => {
    const artifacts = [
      makeArtifact({
        id: "a1",
        source_uri: "context-os/app#1",
        metadata: { github_owner: "context-os", github_repo: "app", object_id: "1" },
        ingested_at: "2026-06-03T10:00:00.000Z",
      }),
      makeArtifact({
        id: "a2",
        source_uri: "context-os/app#2",
        metadata: { github_owner: "context-os", github_repo: "app" },
        ingested_at: "2026-06-03T11:00:00.000Z",
      }),
    ];

    const groups = groupArtifactsBySource(artifacts);

    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe("context-os/app");
    expect(groups[0].artifacts).toHaveLength(2);
    expect(artifactDetailRows(artifacts[0])).toContainEqual(["metadata.object_id", "1"]);
  });

  it("extracts readable event summaries, facts, and links", () => {
    const artifact = makeArtifact({
      title: "Slack decision",
      body: [
        "- **Alice** confirmed the API `field` mapping",
        "- Bob asked QA to wait for the backend deploy",
        "Message link: https://slack.example.test/archives/C1/p1",
      ].join("\n"),
      preview: "",
    });

    const summary = activityEventSummary(artifact);

    expect(summary.preview).toBe("Slack decision");
    expect(summary.detailText).toBe("Slack decision");
    expect(summary.facts).toEqual([
      "**Alice** confirmed the API `field` mapping",
      "Bob asked QA to wait for the backend deploy",
    ]);
    expect(summary.links).toEqual(["https://slack.example.test/archives/C1/p1"]);
    expect(summary.rawText).toContain("**Alice** confirmed");
  });
});

describe("previewText", () => {
  it("cleans markdown and truncates long previews", () => {
    expect(previewText("**Important** `field`", 20)).toBe("Important field");
    expect(previewText("one two three four", 8)).toBe("one two...");
  });
});

describe("previewMarkdownText", () => {
  it("preserves markdown markers while truncating expanded detail text", () => {
    expect(previewMarkdownText("**Important** `field`", 40)).toBe(
      "**Important** `field`",
    );
    expect(previewMarkdownText("**Important** `field`", 14)).toBe(
      "**Important**...",
    );
  });
});

describe("markdownBulletList", () => {
  it("turns source arrays into safe markdown bullet text", () => {
    expect(
      markdownBulletList([
        "**Fact** uses `field_id`",
        "- Existing bullet",
        "1. Existing step",
        "",
        "中文 `字段`",
      ]),
    ).toBe(
      [
        "- **Fact** uses `field_id`",
        "- Existing bullet",
        "1. Existing step",
        "- 中文 `字段`",
      ].join("\n"),
    );
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

function makeMessage(
  id: string,
  createdAt: string,
  mismatchCount: number,
): ChatMessage {
  return {
    id,
    role: "assistant",
    text: "Analysis complete",
    createdAt,
    card: {
      kind: "findings",
      findingsResult: {
        mismatch_count: mismatchCount,
        mismatches: [{ id: `${id}-finding`, summary: `${id} finding` }],
      },
    },
  };
}
