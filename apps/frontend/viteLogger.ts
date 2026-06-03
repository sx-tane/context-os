export function logProxy(label: string) {
  return (proxy: {
    on: (
      event: "proxyReq" | "proxyRes" | "error",
      handler: (...args: any[]) => void,
    ) => void;
  }) => {
    proxy.on("proxyReq", (_proxyReq, req) => {
      if (!proxyLogsEnabled()) return;
      console.log(
        `[proxy] -> ${label} ${req.method} ${req.url} id=${requestID(req)}`,
      );
    });
    proxy.on("proxyRes", (proxyRes, req) => {
      if (!proxyLogsEnabled()) return;
      console.log(
        `[proxy] <- ${label} headers ${proxyRes.statusCode} ${req.method} ${req.url} id=${requestID(req)}`,
      );
      proxyRes.on("end", () => {
        console.log(
          `[proxy] <- ${label} done ${proxyRes.statusCode} ${req.method} ${req.url} id=${requestID(req)}`,
        );
      });
      proxyRes.on("error", (err: Error) => {
        console.log(
          `[proxy] xx ${label} response ${req.method} ${req.url} id=${requestID(req)} ${err.message}`,
        );
      });
    });
    proxy.on("error", (err, req) => {
      if (!proxyLogsEnabled()) return;
      console.log(
        `[proxy] xx ${label} ${req.method} ${req.url} id=${requestID(req)} ${err.message}`,
      );
    });
  };
}

function proxyLogsEnabled(): boolean {
  return process.env.CONTEXTOS_PROXY_LOGS === "1";
}

function requestID(req: { headers: Record<string, string | string[] | undefined> }): string {
  const value = req.headers["x-contextos-request-id"];
  if (Array.isArray(value)) return value[0] ?? "-";
  return value ?? "-";
}
