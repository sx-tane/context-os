jest.mock("$lib/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  demoArtifacts,
  demoChatQueryResult,
  demoFindings,
  demoGraphData,
  demoWorkspaceStatus,
} from "../demoWorkspace";

describe("demoWorkspaceStatus", () => {
  it("exposes a local-only demo workspace with ready source syncs", () => {
    const status = demoWorkspaceStatus();

    expect(status.workspace?.id).toBe("contextos-demo");
    expect(status.syncs).toHaveLength(3);
    expect(status.syncs?.every((sync) => sync.status === "ready")).toBe(true);
  });
});

describe("demoFindings", () => {
  it("returns seeded findings with evidence, confidence, and recommended actions", () => {
    const findings = demoFindings();

    expect(findings.mismatch_count).toBe(2);
    expect(findings.mismatches).toHaveLength(2);
    expect(findings.mismatches?.[0].evidence?.length).toBeGreaterThan(0);
    expect(findings.mismatches?.[0].confidence).toBeGreaterThan(0);
    expect(findings.mismatches?.[0].recommended_action).toContain("Assign");
  });
});

describe("demoGraphData", () => {
  it("keeps graph counts aligned with seeded entities and relationships", () => {
    const graph = demoGraphData();

    expect(graph.workspace_id).toBe("contextos-demo");
    expect(graph.entity_count).toBe(graph.entities.length);
    expect(graph.relationship_count).toBe(graph.relationships?.length);
  });
});

describe("demoArtifacts", () => {
  it("returns recent activity artifacts for multiple connectors", () => {
    const artifacts = demoArtifacts();

    expect(artifacts).toHaveLength(4);
    expect(new Set(artifacts.map((artifact) => artifact.connector))).toEqual(
      new Set(["jira", "github", "slack"]),
    );
  });
});

describe("demoChatQueryResult", () => {
  it("answers finding-oriented prompts with the demo finding context", () => {
    const result = demoChatQueryResult("show refund findings");

    expect(result.workspace_id).toBe("contextos-demo");
    expect(result.intent).toBe("findings");
    expect(result.answer).toContain("requirement gap");
    expect(result.artifact_count).toBe(result.artifacts.length);
  });
});
