import {
  isAnalysisEligibleSource,
  isBroadConnectorScope,
  sourceSetupURI,
  splitAnalysisSources,
} from "../sources/analysisEligibility";

import type { ConnectorKnowledge } from "../types";

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

function makeSource(overrides: Partial<ConnectorKnowledge>): ConnectorKnowledge {
  return {
    connector: "github",
    uri: "github",
    status: "ready",
    ...overrides,
  };
}
