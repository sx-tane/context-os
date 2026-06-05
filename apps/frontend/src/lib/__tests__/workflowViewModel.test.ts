import {
  activityEvidenceType,
  askChatPromptForEvidence,
  basketItemFromAnswerSection,
  basketItemFromArtifact,
  buildAnalysisPreview,
  buildSourceHealth,
  buildWorkspaceSnapshotMarkdown,
  filterActivityArtifacts,
  findingActionLabel,
  findingShareText,
  isFindingVisible,
  mergeBasketItem,
  nextFindingActionStatus,
  uniqueFindings,
} from "../workflow/viewModel";

import type {
  Artifact,
  EvidenceBasketItem,
  FindingsMismatch,
  GraphData,
} from "../types";

describe("buildAnalysisPreview", () => {
  it("marks basket items as included and other eligible sources as available", () => {
    const preview = buildAnalysisPreview({
      readySources: [
        { connector: "github", uri: "github", status: "ready" },
        { connector: "github", uri: "owner/repo", status: "ready" },
      ],
      basketItems: [basketItem({ connector: "slack", uri: "#release" })],
    });

    expect(preview.included.map((row) => row.label)).toEqual(["slack:#release"]);
    expect(preview.available.map((row) => row.label)).toEqual(["github:owner/repo"]);
    expect(preview.skipped.map((row) => row.label)).toEqual(["github:github"]);
    expect(preview.hasBasketSelection).toBe(true);
  });
});

describe("filterActivityArtifacts", () => {
  it("combines connector, source, evidence type, and keyword filters", () => {
    const artifacts = [
      artifact({
        id: "a",
        connector: "slack",
        source_uri: "#release",
        title: "Payment rollout",
        metadata: { evidence_kind: "live_chat_answer" },
      }),
      artifact({
        id: "b",
        connector: "github",
        source_uri: "owner/repo",
        title: "Other evidence",
      }),
    ];

    const filtered = filterActivityArtifacts(artifacts, {
      connector: "slack",
      sourceURI: "release",
      evidenceType: "live_chat_answer",
      keyword: "payment",
    });

    expect(filtered.map((item) => item.id)).toEqual(["a"]);
    expect(activityEvidenceType(filtered[0])).toBe("live_chat_answer");
  });
});

describe("basket helpers", () => {
  it("builds basket items from chat source sections and Activity artifacts", () => {
    const fromSection = basketItemFromAnswerSection({
      source_label: "Jira BKGDEV-8551",
      connector: "jira",
      source_uri: "BKGDEV-8551",
    }, "message-1", new Date("2026-06-04T00:00:00.000Z"));
    const fromArtifact = basketItemFromArtifact(artifact({
      id: "artifact-1",
      connector: "github",
      source_uri: "https://github.com/context-os/app/pull/43",
      metadata: { source_label: "GitHub PR" },
    }), new Date("2026-06-04T00:00:00.000Z"));

    expect(fromSection).toMatchObject({
      id: "jira:BKGDEV-8551",
      origin: "chat",
      messageId: "message-1",
    });
    expect(fromArtifact).toMatchObject({
      id: "github:https://github.com/context-os/app/pull/43",
      origin: "activity",
      artifactId: "artifact-1",
    });
    expect(mergeBasketItem([fromSection!], fromArtifact!)).toHaveLength(2);
  });

  it("builds a source-prefill chat prompt without auto-sending", () => {
    expect(askChatPromptForEvidence("jira", "BKGDEV-8551", "Receipt issuer")).toContain(
      "jira:BKGDEV-8551",
    );
  });
});

describe("buildSourceHealth", () => {
  it("labels broad, concrete, evidence-backed, and error sources", () => {
    const rows = buildSourceHealth({
      codexLoggedIn: true,
      readySources: [
        { connector: "github", uri: "github", status: "ready" },
        { connector: "jira", uri: "BKGDEV-8551", status: "ready" },
        { connector: "slack", uri: "#release", status: "error", error: "reauth" },
      ],
      recentArtifacts: [
        artifact({ connector: "github", source_uri: "owner/repo" }),
      ],
    });

    expect(rows.find((row) => row.label === "github:github")?.status).toBe("broad-chat-only");
    expect(rows.find((row) => row.label === "jira:BKGDEV-8551")?.status).toBe("analysis-ready");
    expect(rows.find((row) => row.label === "github:owner/repo")?.status).toBe("analysis-ready");
    expect(rows.find((row) => row.label === "slack:#release")?.status).toBe("needs-attention");
  });
});

describe("finding workflow helpers", () => {
  it("cycles checklist status and builds copy text", () => {
    const finding: FindingsMismatch = {
      id: "finding-1",
      summary: "Payment type mismatch",
      recommended_action: "Align receipt_issuer enum.",
      evidence: ["jira:BKGDEV-8551"],
    };

    expect(nextFindingActionStatus("open")).toBe("checking");
    expect(nextFindingActionStatus("checking")).toBe("done");
    expect(nextFindingActionStatus("done")).toBe("open");
    expect(findingActionLabel("false_positive")).toBe("false positive");
    expect(isFindingVisible({
      findingId: "finding-1",
      status: "false_positive",
      updatedAt: "2026-06-04T00:00:00.000Z",
    }, "active")).toBe(false);
    expect(isFindingVisible({
      findingId: "finding-1",
      status: "false_positive",
      updatedAt: "2026-06-04T00:00:00.000Z",
    }, "false_positive")).toBe(true);
    expect(findingShareText(finding, {
      findingId: "finding-1",
      status: "checking",
      updatedAt: "2026-06-04T00:00:00.000Z",
    })).toContain("Align receipt_issuer enum");
  });

  it("keeps only the first finding for each stable id", () => {
    const findings: FindingsMismatch[] = [
      { id: "finding-1", summary: "Payment type mismatch" },
      { id: "finding-1", summary: "Payment type mismatch duplicate" },
      { summary: "Fallback summary identity" },
      { summary: "Fallback summary identity" },
    ];

    expect(uniqueFindings(findings)).toEqual([
      { id: "finding-1", summary: "Payment type mismatch" },
      { summary: "Fallback summary identity" },
    ]);
  });
});

describe("buildWorkspaceSnapshotMarkdown", () => {
  it("includes analysis, findings, graph, Activity, and checklist state", () => {
    const preview = buildAnalysisPreview({
      basketItems: [basketItem({ connector: "jira", uri: "BKGDEV-8551" })],
    });
    const markdown = buildWorkspaceSnapshotMarkdown({
      workspacePath: "/workspace",
      preview,
      sourceHealth: [{
        id: "jira:BKGDEV-8551",
        connector: "jira",
        uri: "BKGDEV-8551",
        label: "jira:BKGDEV-8551",
        status: "analysis-ready",
        detail: "Concrete source ready",
      }],
      findings: {
        mismatches: [{ id: "finding-1", summary: "Payment mismatch" }],
        mismatch_count: 1,
      },
      actions: [{
        findingId: "finding-1",
        status: "done",
        updatedAt: "2026-06-04T00:00:00.000Z",
      }],
      graphData: graphData(),
      recentArtifacts: [artifact({ title: "Activity evidence" })],
      basketItems: [basketItem({ connector: "jira", uri: "BKGDEV-8551" })],
    });

    expect(markdown).toContain("# ContextOS Snapshot: /workspace");
    expect(markdown).toContain("[done] Payment mismatch");
    expect(markdown).toContain("Nodes: 2");
    expect(markdown).toContain("Activity evidence");
  });
});

function basketItem(overrides: Partial<EvidenceBasketItem>): EvidenceBasketItem {
  return {
    id: `${overrides.connector ?? "github"}:${overrides.uri ?? "owner/repo"}`,
    connector: "github",
    uri: "owner/repo",
    label: `${overrides.connector ?? "github"}:${overrides.uri ?? "owner/repo"}`,
    origin: "activity",
    addedAt: "2026-06-04T00:00:00.000Z",
    ...overrides,
  };
}

function artifact(overrides: Partial<Artifact>): Artifact {
  return {
    id: "artifact",
    workspace_id: "workspace",
    connector: "slack",
    source_uri: "#release",
    event_type: "message",
    title: "Evidence",
    body: "Payment evidence body",
    preview: "Payment evidence preview",
    content_hash: "hash",
    schema_version: "1",
    ingested_at: "2026-06-04T00:00:00.000Z",
    ...overrides,
  };
}

function graphData(): GraphData {
  return {
    workspace_id: "workspace",
    count: 2,
    entity_count: 2,
    relationship_count: 1,
    entities: [],
    relationships: [],
  };
}
