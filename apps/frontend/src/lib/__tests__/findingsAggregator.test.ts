import {
  aggregateFindings,
  buildFindingsRunSummary,
  type FindingsFailure,
} from "../findings/aggregator";

describe("aggregateFindings", () => {
  it("merges mismatches and sums counts across multiple findings responses", () => {
    const result = aggregateFindings([
      {
        connector: "github",
        uri: "repo",
        mismatch_count: 1,
        event_count: 3,
        entity_count: 4,
        severity_count: { high: 1, medium: 0, low: 0 },
        mismatch_ids: ["m1"],
        mismatches: [{ id: "m1", severity: "high", description: "drift" }],
      },
      {
        connector: "slack",
        uri: "channel",
        mismatch_count: 2,
        event_count: 5,
        entity_count: 6,
        severity_count: { high: 0, medium: 2, low: 0 },
        mismatch_ids: ["m2", "m3"],
        mismatches: [
          { id: "m2", severity: "medium", description: "gap" },
          { id: "m3", severity: "medium", description: "risk" },
        ],
      },
    ]);

    expect(result?.connector).toBe("multiple");
    expect(result?.mismatch_count).toBe(3);
    expect(result?.event_count).toBe(8);
    expect(result?.entity_count).toBe(10);
    expect(result?.severity_count).toEqual({ high: 1, medium: 2, low: 0 });
    expect(result?.mismatches).toHaveLength(3);
    expect(result?.mismatch_ids).toEqual(["m1", "m2", "m3"]);
  });

  it("returns a zero-finding aggregate when analysis succeeds without mismatch signals", () => {
    const result = aggregateFindings([
      { connector: "slack", uri: "channel", mismatch_count: 0, event_count: 7, entity_count: 2, mismatches: [] },
    ]);

    expect(result?.mismatch_count).toBe(0);
    expect(result?.summary).toContain("no mismatch signals detected");
    expect(result?.event_count).toBe(7);
    expect(result?.entity_count).toBe(2);
  });
});

describe("buildFindingsRunSummary", () => {
  it("describes a successful zero-finding run with source, event, and entity counts", () => {
    const message = buildFindingsRunSummary({
      sourceCount: 1,
      completedCount: 1,
      result: { mismatch_count: 0, event_count: 4, entity_count: 3 },
      failures: [],
    });

    expect(message).toContain("Analysis ran, no mismatch signals detected");
    expect(message).toContain("Sources: 1");
    expect(message).toContain("Events: 4");
    expect(message).toContain("Entities: 3");
  });

  it("includes per-source failures when one source fails and another succeeds", () => {
    const failures: FindingsFailure[] = [
      { connector: "jira", uri: "project", message: "unauthorized" },
    ];
    const message = buildFindingsRunSummary({
      sourceCount: 2,
      completedCount: 1,
      result: { mismatch_count: 1, event_count: 2, entity_count: 2 },
      failures,
    });

    expect(message).toContain("Analysis complete for 1/2 concrete sources");
    expect(message).toContain("Found 1 finding");
    expect(message).toContain("Failed:");
    expect(message).toContain("jira:project - unauthorized");
  });

  it("reports skipped chat-only scopes separately from failures", () => {
    const message = buildFindingsRunSummary({
      sourceCount: 2,
      analysisSourceCount: 0,
      completedCount: 0,
      result: null,
      failures: [],
      skipped: [
        {
          connector: "github",
          uri: "github",
          reason: "chat-only live connector scope",
        },
      ],
    });

    expect(message).toContain("Analysis skipped");
    expect(message).toContain("Skipped chat-only scopes");
    expect(message).toContain("github:github - chat-only live connector scope");
    expect(message).not.toContain("Failed:");
  });
});
