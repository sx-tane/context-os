import type { IngestProvider, IngestResult } from "$lib/types";
import { postIngest, streamCodexIngest } from "$lib/api";

type ConnectorKind = "github" | "slack";

interface IngestRunnerOptions {
  connector: ConnectorKind;
  uri: string;
  token?: string;
  provider: IngestProvider;
  setLoading: (loading: boolean) => void;
  setError: (message: string) => void;
  setResult: (result: IngestResult | null) => void;
  setLiveLog: (log: string | ((current: string) => string)) => void;
  setElapsed: (elapsed: number | ((current: number) => number)) => void;
  signal?: AbortSignal;
}

export async function runConnectorIngest(options: IngestRunnerOptions): Promise<void> {
  let timer: ReturnType<typeof setInterval> | null = null;

  options.setLoading(true);
  options.setError("");
  options.setResult(null);
  options.setLiveLog("");
  options.setElapsed(0);

  try {
    if (options.provider === "codex") {
      timer = setInterval(() => {
        options.setElapsed((current) => current + 1);
      }, 1000);

      await streamCodexIngest(
        options.connector,
        { uri: options.uri, token: options.token || undefined, provider: "codex" },
        {
          onLog: (line) => options.setLiveLog((current) => current + line + "\n"),
          onStatus: (seconds) => options.setElapsed(seconds),
          onResult: (body) => options.setResult(body),
          onError: (message) => options.setError(message),
        },
        { signal: options.signal },
      );
      return;
    }

    const res = await postIngest(
      options.connector,
      {
        uri: options.uri,
        token: options.token || undefined,
        provider: options.provider,
      },
      { signal: options.signal },
    );
    if (!res.ok) {
      options.setError(res.body?.message ?? `Request failed with status ${res.status}`);
      return;
    }
    options.setResult(res.body);
  } catch (err) {
    options.setError(err instanceof Error ? err.message : String(err));
  } finally {
    if (timer) clearInterval(timer);
    options.setLoading(false);
  }
}
