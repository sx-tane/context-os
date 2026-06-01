import {
  probeService,
  getJSON,
  postIngest,
  postFindings,
  postFilesystemUpload,
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
