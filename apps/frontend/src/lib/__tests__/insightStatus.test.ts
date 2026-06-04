import { buildInsightStatus } from "../insights/status";
import type {
  Artifact,
  ConnectorKnowledge,
  FindingsResult,
  GraphData,
} from "../types";

describe("buildInsightStatus", () => {
  it("reports no concrete sources when the workspace has no ready sources", () => {
    const status = buildInsightStatus({});

    expect(status.concreteSourceCount).toBe(0);
    expect(status.chatOnlySourceCount).toBe(0);
    expect(status.findingsState).toBe("no_concrete_sources");
    expect(status.findingsMessage).toContain("Add a concrete source");
  });

  it("counts connector-only live scopes as chat-only instead of analysis-ready", () => {
    const status = buildInsightStatus({
      readySources: [
        readySource("github", "github"),
        readySource("slack", "slack"),
      ],
    });

    expect(status.concreteSourceCount).toBe(0);
    expect(status.chatOnlySourceCount).toBe(2);
    expect(status.findingsState).toBe("no_concrete_sources");
    expect(status.sourceScopeLabel).toBe("0 concrete sources, 2 chat-only scopes");
    expect(status.chatOnlySources.map((source) => source.connector)).toEqual([
      "github",
      "slack",
    ]);
  });

  it("marks concrete sources with graph context as not run before manual analysis", () => {
    const status = buildInsightStatus({
      readySources: [readySource("github", "owner/repo")],
      graphData: graphData({ nodes: 3, links: 2 }),
    });

    expect(status.concreteSourceCount).toBe(1);
    expect(status.hasGraphContext).toBe(true);
    expect(status.findingsState).toBe("not_run");
    expect(status.findingsMessage).toBe("Graph has context, findings not run yet.");
  });

  it("marks findings stale when Activity evidence is newer than the last analysis", () => {
    const status = buildInsightStatus({
      readySources: [readySource("jira", "BKGDEV-8466")],
      recentArtifacts: [
        artifact("2026-06-04T08:00:00.000Z"),
        artifact("2026-06-04T11:00:00.000Z"),
      ],
      lastFindings: findings({ mismatchCount: 0 }),
      lastAnalysisAt: "2026-06-04T09:00:00.000Z",
    });

    expect(status.latestActivityAt).toBe("2026-06-04T11:00:00.000Z");
    expect(status.findingsState).toBe("stale");
    expect(status.findingsMessage).toContain("Activity has newer evidence");
  });

  it("marks findings current after analysis covers the latest Activity evidence", () => {
    const status = buildInsightStatus({
      readySources: [readySource("filesystem", "/tmp/workspace")],
      recentArtifacts: [artifact("2026-06-04T08:00:00.000Z")],
      graphData: graphData({ nodes: 4, links: 3 }),
      lastFindings: findings({ mismatchCount: 1 }),
      lastAnalysisAt: "2026-06-04T09:00:00.000Z",
    });

    expect(status.findingsState).toBe("current");
    expect(status.findingCount).toBe(1);
    expect(status.footerLabel).toContain("Activity:");
    expect(status.footerLabel).toContain("Graph:");
    expect(status.footerLabel).toContain("Findings:");
  });
});

function readySource(
  connector: ConnectorKnowledge["connector"],
  uri: string,
): ConnectorKnowledge {
  return {
    connector,
    uri,
    status: "ready",
  };
}

function artifact(ingestedAt: string): Artifact {
  return {
    id: `artifact-${ingestedAt}`,
    workspace_id: "workspace",
    connector: "github",
    source_uri: "owner/repo",
    event_type: "message",
    title: "Evidence",
    body: "Evidence body",
    preview: "Evidence preview",
    content_hash: "hash",
    schema_version: "1",
    ingested_at: ingestedAt,
  };
}

function graphData({ nodes, links }: { nodes: number; links: number }): GraphData {
  return {
    workspace_id: "workspace",
    count: nodes,
    entity_count: nodes,
    relationship_count: links,
    entities: [],
    relationships: [],
  };
}

function findings({ mismatchCount }: { mismatchCount: number }): FindingsResult {
  return {
    mismatch_count: mismatchCount,
    mismatches: Array.from({ length: mismatchCount }, (_, index) => ({
      id: `mismatch-${index}`,
    })),
  };
}
