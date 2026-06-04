jest.mock("$lib/workspace/projectStore", () => ({
  DEMO_WORKSPACE_PATH: "contextos-demo",
}));

import {
  demoArtifacts,
  demoChatQueryResult,
  demoFindings,
  demoGraphData,
  demoPlanningTourResult,
  demoWorkspaceStatus,
} from "../chat/demoWorkspace";

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

    expect(artifacts.length).toBeGreaterThanOrEqual(6);
    expect(new Set(artifacts.map((artifact) => artifact.connector))).toEqual(
      new Set(["demo", "jira", "github", "slack"]),
    );
    expect(artifacts[0].title).toContain("Planning mode");
  });
});

describe("demoChatQueryResult", () => {
  it("answers finding-oriented prompts with the demo finding context", () => {
    const result = demoChatQueryResult("show refund findings");

    expect(result.workspace_id).toBe("contextos-demo");
    expect(result.intent).toBe("findings");
    expect(result.answer).toContain("requirement gap");
    expect(result.answer_sections?.[0].source_label).toContain("Finding");
    expect(result.artifact_count).toBe(result.artifacts.length);
  });

  it("answers planning prompts with source-card demo notes", () => {
    const result = demoChatQueryResult("show planning mode functions");

    expect(result.intent).toBe("demo_tour");
    expect(result.answer).toContain("Demo planning mode");
    expect(result.answer_sections?.[0].source_label).toBe(
      "Demo · Planning notes",
    );
    expect(result.answer_sections?.[0].facts?.join("\n")).toContain(
      "Agent chat",
    );
  });

  it("answers source-card prompts with grouped connector sections", () => {
    const result = demoChatQueryResult("show source cards");

    expect(result.intent).toBe("status");
    expect(result.answer_sections?.map((section) => section.connector)).toEqual(
      ["jira", "github", "slack"],
    );
  });

  it("answers activity cleanup prompts without mutating demo data", () => {
    const result = demoChatQueryResult("activity cleanup");

    expect(result.summary).toContain("cleanup");
    expect(result.answer_sections?.[0].open_items?.join("\n")).toContain(
      "Cleanup never runs automatically",
    );
  });
});

describe("demoPlanningTourResult", () => {
  it("builds the seeded demo chat result with structured planning notes", () => {
    const result = demoPlanningTourResult();

    expect(result.workspace_id).toBe("contextos-demo");
    expect(result.answer_sections).toHaveLength(1);
    expect(result.answer_sections?.[0].coding_notes?.join("\n")).toContain(
      "Ask for planning mode",
    );
    expect(result.artifact_count).toBe(result.artifacts.length);
  });
});
