import {
  buildAnalysisSources,
  isAnalysisEligibleSource,
  isBroadConnectorScope,
  maxDerivedAnalysisSources,
  sourceSetupURI,
  splitAnalysisSources,
} from "../sources/analysisEligibility";

import type {
  AnswerSection,
  Artifact,
  ChatQueryResult,
  ConnectorKnowledge,
} from "../types";

describe("sourceSetupURI", () => {
  it("preserves typed live URIs and only uses connector scope when no URI is typed", () => {
    expect(sourceSetupURI("github", "owner/repo", true)).toBe("owner/repo");
    expect(sourceSetupURI("github", "", true)).toBe("github");
    expect(sourceSetupURI("github", "", false)).toBe("");
    expect(sourceSetupURI("filesystem", "docs/", true)).toBe("docs/");
    expect(sourceSetupURI("filesystem", "", true)).toBe("");
  });
});

describe("analysis source eligibility", () => {
  it("treats connector-only live scopes as chat-only and concrete sources as eligible", () => {
    const broad = makeSource({ connector: "github", uri: "github" });
    const repo = makeSource({ connector: "github", uri: "owner/repo" });
    const local = makeSource({ connector: "filesystem", uri: "docs/" });

    expect(isBroadConnectorScope(broad)).toBe(true);
    expect(isAnalysisEligibleSource(broad)).toBe(false);
    expect(isAnalysisEligibleSource(repo)).toBe(true);
    expect(isAnalysisEligibleSource(local)).toBe(true);

    const split = splitAnalysisSources([broad, repo, local]);
    expect(split.eligible.map((source) => source.uri)).toEqual(["owner/repo", "docs/"]);
    expect(split.skipped).toEqual([
      {
        connector: "github",
        uri: "github",
        reason: "chat-only live connector scope",
      },
    ]);
  });
});

describe("buildAnalysisSources", () => {
  it("derives concrete Jira issue sources from chat answer sections", () => {
    const built = buildAnalysisSources({
      readySources: [makeSource({ connector: "jira", uri: "jira" })],
      lastChatResult: chatResult({
        answer_sections: [
          {
            source_label: "Jira issue BKGDEV-8551",
            connector: "jira",
            source_uri: "BKGDEV-8551",
          },
        ],
      }),
    });

    expect(built.eligible.map(sourceKey)).toEqual(["jira:BKGDEV-8551"]);
    expect(built.derived.map(sourceKey)).toEqual(["jira:BKGDEV-8551"]);
    expect(built.skipped.map(sourceKey)).toEqual(["jira:jira"]);
  });

  it("derives GitHub repo, issue, and PR URLs when source_uri is missing", () => {
    const built = buildAnalysisSources({
      lastChatResult: chatResult({
        answer_sections: [
          {
            source_label: "GitHub repo",
            links: ["https://github.com/context-os/app"],
          },
          {
            source_label: "GitHub issue",
            links: ["https://github.com/context-os/app/issues/42"],
          },
          {
            source_label: "GitHub PR",
            connector: "github",
            links: ["https://github.com/context-os/app/pull/43"],
          },
        ],
      }),
    });

    expect(built.eligible.map(sourceKey)).toEqual([
      "github:https://github.com/context-os/app",
      "github:https://github.com/context-os/app/issues/42",
      "github:https://github.com/context-os/app/pull/43",
    ]);
  });

  it("derives Slack channel and thread links from chat source evidence", () => {
    const built = buildAnalysisSources({
      lastChatResult: chatResult({
        answer_sections: [
          {
            source_label: "Slack channel",
            connector: "slack",
            source_uri: "#receipt-issuer",
          },
          {
            source_label: "Slack thread",
            links: [
              "https://acme.slack.com/archives/C123/p1717449300000000",
            ],
          },
        ],
      }),
    });

    expect(built.eligible.map(sourceKey)).toEqual([
      "slack:#receipt-issuer",
      "slack:https://acme.slack.com/archives/C123/p1717449300000000",
    ]);
  });

  it("derives Google Drive document and file URLs from chat source evidence", () => {
    const built = buildAnalysisSources({
      lastChatResult: chatResult({
        answer_sections: [
          {
            source_label: "Drive doc",
            links: ["https://docs.google.com/document/d/doc-123/edit"],
          },
          {
            source_label: "Drive file",
            source_uri: "https://drive.google.com/file/d/file-456/view",
          },
        ],
      }),
    });

    expect(built.eligible.map(sourceKey)).toEqual([
      "googledrive:https://docs.google.com/document/d/doc-123/edit",
      "googledrive:https://drive.google.com/file/d/file-456/view",
    ]);
  });

  it("uses concrete Activity artifacts created from live chat evidence", () => {
    const built = buildAnalysisSources({
      readySources: [makeSource({ connector: "github", uri: "github" })],
      recentArtifacts: [
        artifact({
          connector: "googledrive",
          source_uri: "https://drive.google.com/file/d/file-789/view",
          metadata: {
            evidence_kind: "live_chat_answer",
          },
        }),
      ],
    });

    expect(built.eligible.map(sourceKey)).toEqual([
      "googledrive:https://drive.google.com/file/d/file-789/view",
    ]);
    expect(built.derived).toHaveLength(1);
    expect(built.skipped.map(sourceKey)).toEqual(["github:github"]);
  });

  it("deduplicates sources and caps derived evidence to the safe source-card limit", () => {
    const sections: AnswerSection[] = Array.from(
      { length: maxDerivedAnalysisSources + 2 },
      (_, index) => ({
        source_label: `Jira ${index}`,
        connector: "jira",
        source_uri: `BKGDEV-${index + 1}`,
      }),
    );
    const built = buildAnalysisSources({
      readySources: [makeSource({ connector: "jira", uri: "BKGDEV-1" })],
      lastChatResult: chatResult({ answer_sections: sections }),
    });

    expect(built.derived).toHaveLength(maxDerivedAnalysisSources);
    expect(built.eligible[0]).toMatchObject({
      connector: "jira",
      uri: "BKGDEV-1",
    });
    expect(new Set(built.eligible.map(sourceKey)).size).toBe(
      built.eligible.length,
    );
  });

  it("uses basket sources only while preserving other eligible sources as available", () => {
    const built = buildAnalysisSources({
      readySources: [
        makeSource({ connector: "github", uri: "github" }),
        makeSource({ connector: "github", uri: "owner/repo" }),
      ],
      lastChatResult: chatResult({
        answer_sections: [
          {
            source_label: "Jira",
            connector: "jira",
            source_uri: "BKGDEV-8551",
          },
        ],
      }),
      basketItems: [
        {
          id: "slack:#release",
          connector: "slack",
          uri: "#release",
          label: "Release",
          origin: "activity",
          addedAt: "2026-06-04T00:00:00.000Z",
        },
      ],
    });

    expect(built.eligible.map(sourceKey)).toEqual(["slack:#release"]);
    expect(built.basket.map(sourceKey)).toEqual(["slack:#release"]);
    expect(built.available.map(sourceKey)).toEqual([
      "github:owner/repo",
      "jira:BKGDEV-8551",
    ]);
    expect(built.skipped.map(sourceKey)).toEqual(["github:github"]);
  });
});

function makeSource(overrides: Partial<ConnectorKnowledge>): ConnectorKnowledge {
  return {
    connector: "github",
    uri: "github",
    status: "ready",
    ...overrides,
  };
}

function sourceKey(source: Pick<ConnectorKnowledge, "connector" | "uri">) {
  return `${source.connector}:${source.uri}`;
}

function chatResult(overrides: Partial<ChatQueryResult>): ChatQueryResult {
  return {
    intent: "artifacts",
    workspace_id: "workspace",
    workspace_path: "workspace",
    provider: "codex",
    answer: "Answer",
    summary: "Summary",
    artifact_count: 0,
    artifacts: [],
    ...overrides,
  };
}

function artifact(overrides: Partial<Artifact>): Artifact {
  return {
    id: "artifact",
    workspace_id: "workspace",
    connector: "github",
    source_uri: "owner/repo",
    event_type: "document.ingested",
    title: "Evidence",
    body: "Evidence body",
    preview: "Evidence preview",
    content_hash: "hash",
    schema_version: "1",
    ingested_at: "2026-06-04T00:00:00.000Z",
    ...overrides,
  };
}
