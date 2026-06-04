import { applyWorkspaceSyncsToConnectors } from "$lib/workspace/statusMapping";

describe("applyWorkspaceSyncsToConnectors", () => {
  it("keeps connected source rows ready without assigning empty event counts", () => {
    const result = applyWorkspaceSyncsToConnectors(
      [{
        connector: "github",
        uri: "owner/repo",
        status: "ready",
        eventCount: 4,
      }],
      [{
        connector: "github",
        source_uri: "owner/repo",
        status: "connected",
        event_count: 0,
      }],
    );

    expect(result.changed).toBe(true);
    expect(result.connectors[0]).toEqual(expect.objectContaining({
      connector: "github",
      uri: "owner/repo",
      status: "ready",
      eventCount: undefined,
      error: undefined,
    }));
  });

  it("keeps pending source rows ready when the workspace database still has the source", () => {
    const result = applyWorkspaceSyncsToConnectors(
      [{
        connector: "slack",
        uri: "#team",
        status: "ready",
      }],
      [{
        connector: "slack",
        source_uri: "#team",
        status: "pending",
        event_count: 0,
      }],
    );

    expect(result.changed).toBe(false);
    expect(result.connectors[0]).toEqual(expect.objectContaining({
      connector: "slack",
      uri: "#team",
      status: "ready",
      eventCount: undefined,
      error: undefined,
    }));
  });

  it("surfaces backend error rows as connector errors", () => {
    const result = applyWorkspaceSyncsToConnectors(
      [{
        connector: "jira",
        uri: "PROJ",
        status: "ready",
      }],
      [{
        connector: "jira",
        source_uri: "PROJ",
        status: "error",
        event_count: 0,
        last_error: "token expired",
      }],
    );

    expect(result.changed).toBe(true);
    expect(result.connectors[0]).toEqual(expect.objectContaining({
      connector: "jira",
      uri: "PROJ",
      status: "error",
      error: "token expired",
    }));
  });

  it("marks uploaded filesystem sources ready when the DB confirms events under a different stored URI", () => {
    const result = applyWorkspaceSyncsToConnectors(
      [{
        connector: "filesystem",
        uri: "20260604_FOOMA様.pdf",
        status: "ingesting",
      }],
      [{
        connector: "filesystem",
        source_uri: "storage/raw/uploads/upload-id/20260604_FOOMA様.pdf",
        status: "ready",
        event_count: 1,
      }],
    );

    expect(result.changed).toBe(true);
    expect(result.connectors[0]).toEqual(expect.objectContaining({
      connector: "filesystem",
      status: "ready",
      eventCount: 1,
      error: undefined,
    }));
  });
});
