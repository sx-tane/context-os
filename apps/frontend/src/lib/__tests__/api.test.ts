import {
  probeService,
  getJSON,
  postIngest,
  postFindings,
  postFilesystemUpload,
  getCodexSources,
  getWorkspaces,
  getArtifacts,
  postChatQuery,
  getGraphData,
  upsertWorkspace,
  getWorkspaceStatus,
} from "../api";

// makeResponse builds a minimal fetch Response mock for readJSON (calls res.text()).
function makeResponse(body: unknown, ok: boolean, status = 200): Response {
  return {
    ok,
    status,
    text: jest
      .fn()
      .mockResolvedValue(
        typeof body === "string" ? body : JSON.stringify(body),
      ),
  } as unknown as Response;
}

const fetchMock = jest.fn<
  Promise<Response>,
  [RequestInfo | URL, RequestInit?]
>();
(global as unknown as Record<string, unknown>).fetch = fetchMock;

beforeEach(() => {
  fetchMock.mockReset();
});

// ---- probeService ----

describe("probeService", () => {
  it("returns 'ok' when the health endpoint responds with 2xx", async () => {
    fetchMock.mockResolvedValue({ ok: true } as Response);
    expect(await probeService("http://api")).toBe("ok");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://api/health",
      expect.objectContaining({ signal: expect.any(Object) }),
    );
  });

  it("returns 'unreachable' when the health endpoint responds with non-2xx", async () => {
    fetchMock.mockResolvedValue({ ok: false } as Response);
    expect(await probeService("http://api")).toBe("unreachable");
  });

  it("returns 'unreachable' when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));
    expect(await probeService("http://api")).toBe("unreachable");
  });
});

// ---- getJSON ----

describe("getJSON", () => {
  it("returns parsed body when response is 2xx", async () => {
    fetchMock.mockResolvedValue(makeResponse({ key: "value" }, true));
    const result = await getJSON<{ key: string }>("/path");
    expect(result).toEqual({ key: "value" });
  });

  it("returns null when response is non-2xx", async () => {
    fetchMock.mockResolvedValue({ ok: false } as Response);
    expect(await getJSON("/path")).toBeNull();
  });

  it("returns null when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));
    expect(await getJSON("/path")).toBeNull();
  });
});

// ---- postIngest ----

describe("postIngest", () => {
  it("returns ok:true with body when response is 2xx", async () => {
    const body = { connector: "github", events: [] };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await postIngest("github", {
      uri: "github://owner/repo",
      provider: "token",
    });
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.body).toEqual(body);
  });

  it("returns ok:false with error body when response is non-2xx", async () => {
    fetchMock.mockResolvedValue(
      makeResponse({ message: "unauthorized" }, false, 401),
    );
    const result = await postIngest("github", {
      uri: "github://owner/repo",
      provider: "token",
    });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.body.message).toBe("unauthorized");
  });

  it("sends connector, URI, and provider in the request body", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await postIngest("slack", { uri: "slack://team/C123", provider: "token" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/slack/ingest",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ uri: "slack://team/C123", provider: "token" }),
      }),
    );
  });
});

// ---- postFindings ----

describe("postFindings", () => {
  it("returns ok:true with body when response is 2xx", async () => {
    const body = { role: "pmo", mismatch_count: 1 };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await postFindings({
      connector: "filesystem",
      uri: "inline.txt",
      content: "frontend expects refundStatus but backend exposes missingRefundState",
      role: "pmo",
    });
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.body).toEqual(body);
  });

  it("returns ok:false with error body when response is non-2xx", async () => {
    fetchMock.mockResolvedValue(
      makeResponse({ message: "invalid_request" }, false, 400),
    );
    const result = await postFindings({ connector: "filesystem" });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.body.message).toBe("invalid_request");
  });

  it("posts to /api/presentation/findings", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await postFindings({ connector: "filesystem", uri: "inline.txt" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/presentation/findings",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

// ---- postFilesystemUpload ----

describe("postFilesystemUpload", () => {
  it("returns ok:true with body when upload succeeds", async () => {
    const body = { connector: "filesystem", events: [] };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await postFilesystemUpload(new FormData());
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.body).toEqual(body);
  });

  it("returns ok:false when upload fails", async () => {
    fetchMock.mockResolvedValue(
      makeResponse({ error: "bad request" }, false, 400),
    );
    const result = await postFilesystemUpload(new FormData());
    expect(result.ok).toBe(false);
  });

  it("posts to /api/filesystem/upload", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await postFilesystemUpload(new FormData());
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/filesystem/upload",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

// ---- getCodexSources ----

describe("getCodexSources", () => {
  it("returns Codex source list when response is 2xx", async () => {
    const body = {
      connector: "github",
      provider: "codex",
      sources: [
        {
          id: "owner/repo",
          label: "owner/repo",
          uri: "owner/repo",
          kind: "repository",
          connector: "github",
        },
      ],
    };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    await expect(getCodexSources("github")).resolves.toEqual(body);
  });

  it("fetches connector source list through /api/codex/sources", async () => {
    fetchMock.mockResolvedValue(makeResponse({ sources: [] }, true, 200));
    await getCodexSources("slack");
    expect(fetchMock).toHaveBeenCalledWith("/api/codex/sources?connector=slack");
  });

  it("returns null when source discovery fails", async () => {
    fetchMock.mockResolvedValue(makeResponse({ message: "failed" }, false, 502));
    await expect(getCodexSources("github")).resolves.toBeNull();
  });
});

// ---- getWorkspaces ----

describe("getWorkspaces", () => {
  it("returns workspace records when response is 2xx", async () => {
    const body = { count: 1, workspaces: [{ id: "ws1", name: "Main", path: "/workspace" }] };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await getWorkspaces();
    expect(result).toEqual(body.workspaces);
  });

  it("normalizes legacy PascalCase workspace records", async () => {
    fetchMock.mockResolvedValue(makeResponse({ count: 1, workspaces: [{ ID: "ws1", Name: "Main", Path: " /workspace " }] }, true, 200));
    const result = await getWorkspaces();
    expect(result).toEqual([{ id: "ws1", name: "Main", path: "/workspace" }]);
  });

  it("fetches /api/workspace", async () => {
    fetchMock.mockResolvedValue(makeResponse({ workspaces: [] }, true, 200));
    await getWorkspaces();
    expect(fetchMock).toHaveBeenCalledWith("/api/workspace");
  });

  it("returns an empty list when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));
    expect(await getWorkspaces()).toEqual([]);
  });
});

// ---- getArtifacts ----

describe("getArtifacts", () => {
  it("returns ArtifactList when response is 2xx", async () => {
    const body = { workspace_id: "ws1", count: 1, artifacts: [{ id: "evt1", connector: "slack" }] };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await getArtifacts({ workspace_id: "ws1", connector: "slack", limit: 5 });
    expect(result).toEqual(body);
  });

  it("encodes all query parameters", async () => {
    fetchMock.mockResolvedValue(makeResponse({ artifacts: [] }, true, 200));
    await getArtifacts({
      workspace_id: "/workspace",
      connector: "slack",
      source_uri: "slack://team/C123",
      q: "today messages",
      since: "2025-01-01T00:00:00Z",
      until: "2025-01-02T00:00:00Z",
      limit: 10,
    });
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("workspace_id=%2Fworkspace");
    expect(url).toContain("connector=slack");
    expect(url).toContain("source_uri=slack%3A%2F%2Fteam%2FC123");
    expect(url).toContain("q=today+messages");
    expect(url).toContain("since=2025-01-01T00%3A00%3A00Z");
    expect(url).toContain("until=2025-01-02T00%3A00%3A00Z");
    expect(url).toContain("limit=10");
  });

  it("returns null when response is non-2xx", async () => {
    fetchMock.mockResolvedValue({ ok: false } as Response);
    expect(await getArtifacts({ workspace_id: "ws1" })).toBeNull();
  });
});

// ---- postChatQuery ----

describe("postChatQuery", () => {
  it("returns ok:true with body when response is 2xx", async () => {
    const body = { intent: "artifacts", answer: "Found 1 artifact", artifact_count: 1, artifacts: [] };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await postChatQuery({ workspace_id: "ws1", message: "give me today slack messages" });
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.body).toEqual(body);
  });

  it("returns ok:false with error body when response is non-2xx", async () => {
    fetchMock.mockResolvedValue(makeResponse({ message: "workspace_id required" }, false, 400));
    const result = await postChatQuery({ workspace_id: "", message: "status" });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.body.message).toBe("workspace_id required");
  });

  it("posts to /api/chat/query with JSON body", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await postChatQuery({ workspace_id: "ws1", message: "today jira tickets", limit: 20 });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/chat/query",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ workspace_id: "ws1", message: "today jira tickets", limit: 20 }),
      }),
    );
  });
});

// ---- getGraphData ----

describe("getGraphData", () => {
  it("returns GraphData when response is 2xx", async () => {
    const body = {
      workspace_id: "ws1",
      count: 2,
      entity_count: 2,
      relationship_count: 1,
      entities: [{ id: "e1", name: "Auth", type: "service", confidence: 0.9 }],
      relationships: [{ id: "r1", from_id: "e1", to_id: "e2", kind: "service_depends_on", confidence: 0.8 }],
    };
    fetchMock.mockResolvedValue(makeResponse(body, true, 200));
    const result = await getGraphData("/workspace");
    expect(result).toEqual(body);
  });

  it("sends workspace_id as query parameter", async () => {
    fetchMock.mockResolvedValue(makeResponse({ count: 0, entities: [] }, true, 200));
    await getGraphData("/my/project");
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("workspace_id=%2Fmy%2Fproject");
  });

  it("sends entity_type query parameter when provided", async () => {
    fetchMock.mockResolvedValue(makeResponse({ count: 0, entities: [] }, true, 200));
    await getGraphData("/proj", "feature");
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("entity_type=feature");
  });

  it("returns null when response is non-2xx", async () => {
    fetchMock.mockResolvedValue({ ok: false } as Response);
    expect(await getGraphData("/workspace")).toBeNull();
  });

  it("returns null when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));
    expect(await getGraphData("/workspace")).toBeNull();
  });
});

// ---- upsertWorkspace ----

describe("upsertWorkspace", () => {
  it("returns WorkspaceRecord when response is 2xx", async () => {
    const ws = { id: "ws1", name: "My Project", path: "/proj" };
    fetchMock.mockResolvedValue(makeResponse(ws, true, 200));
    const result = await upsertWorkspace("/proj", "My Project");
    expect(result).toEqual(ws);
  });

  it("normalizes legacy PascalCase upsert responses", async () => {
    fetchMock.mockResolvedValue(makeResponse({ ID: "ws1", Name: "My Project", Path: "/proj" }, true, 200));
    const result = await upsertWorkspace("/proj", "My Project");
    expect(result).toEqual({ id: "ws1", name: "My Project", path: "/proj" });
  });

  it("posts to /api/workspace/upsert", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await upsertWorkspace("/proj", "My Project");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/workspace/upsert",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("returns null when response is non-2xx", async () => {
    fetchMock.mockResolvedValue({ ok: false } as Response);
    expect(await upsertWorkspace("/proj", "name")).toBeNull();
  });
});

// ---- getWorkspaceStatus ----

describe("getWorkspaceStatus", () => {
  it("returns WorkspaceStatus when response is 2xx", async () => {
    const status = { event_count: 5, entity_count: 3, mismatch_count: 1 };
    fetchMock.mockResolvedValue(makeResponse(status, true, 200));
    const result = await getWorkspaceStatus("/proj");
    expect(result).toEqual(status);
  });

  it("normalizes legacy PascalCase workspace status responses", async () => {
    fetchMock.mockResolvedValue(makeResponse({ Workspace: { ID: "ws1", Name: "Main", Path: "/workspace" }, EventCount: 5, Syncs: [{ Connector: "github", SourceURI: "github://repo", EventCount: 2 }] }, true, 200));
    const result = await getWorkspaceStatus("/workspace");
    expect(result).toEqual({
      workspace: { id: "ws1", name: "Main", path: "/workspace" },
      event_count: 5,
      syncs: [{ connector: "github", source_uri: "github://repo", event_count: 2 }],
    });
  });

  it("encodes workspace path in query string", async () => {
    fetchMock.mockResolvedValue(makeResponse({}, true, 200));
    await getWorkspaceStatus("/my/project");
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("path=%2Fmy%2Fproject");
  });

  it("returns null when fetch throws", async () => {
    fetchMock.mockRejectedValue(new Error("network"));
    expect(await getWorkspaceStatus("/proj")).toBeNull();
  });
});
