import type { ConnectorKind, IngestProvider, IngestResult } from "$lib/types";
import { postIngest, streamCodexIngest } from "$lib/api";

interface IngestRunnerOptions {
  connector: ConnectorKind;
  uri: string;
  token?: string;
  content?: string;
  cursor?: string;
  metadata?: Record<string, string>;
  // SharePoint-specific top-level fields
  tenant_id?: string;
  client_id?: string;
  client_secret?: string;
  provider: IngestProvider;
  setLoading: (loading: boolean) => void;
  setError: (message: string) => void;
  setResult: (result: IngestResult | null) => void;
  setLiveLog: (log: string | ((current: string) => string)) => void;
  setElapsed: (elapsed: number | ((current: number) => number)) => void;
  isCurrent?: () => boolean;
  signal?: AbortSignal;
}

export async function runConnectorIngest(
  options: IngestRunnerOptions,
): Promise<void> {
  let timer: ReturnType<typeof setInterval> | null = null;
  const setIfCurrent = <T>(setter: (value: T) => void, value: T) => {
    if (options.isCurrent?.() === false) return;
    setter(value);
  };

  setIfCurrent(options.setLoading, true);
  setIfCurrent(options.setError, "");
  setIfCurrent(options.setResult, null);
  setIfCurrent(options.setLiveLog, "");
  setIfCurrent(options.setElapsed, 0);

  try {
    if (options.provider === "codex") {
      if (!supportsCodex(options.connector)) {
        setIfCurrent(
          options.setError,
          "Codex streaming is currently available for GitHub, Jira, and Slack connectors.",
        );
        return;
      }

      timer = setInterval(() => {
        setIfCurrent(options.setElapsed, (current) => current + 1);
      }, 1000);

      await streamCodexIngest(
        options.connector,
        {
          uri: options.uri,
          token: options.token || undefined,
          provider: "codex",
        },
        {
          onLog: (line) =>
            setIfCurrent(
              options.setLiveLog,
              (current) => current + line + "\n",
            ),
          onStatus: (seconds) => setIfCurrent(options.setElapsed, seconds),
          onResult: (body) => setIfCurrent(options.setResult, body),
          onError: (message) => setIfCurrent(options.setError, message),
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
        content: options.content || undefined,
        cursor: options.cursor || undefined,
        metadata: cleanMetadata(options.metadata),
        provider: options.provider,
        tenant_id: options.tenant_id || undefined,
        client_id: options.client_id || undefined,
        client_secret: options.client_secret || undefined,
      },
      { signal: options.signal },
    );
    if (!res.ok) {
      setIfCurrent(
        options.setError,
        res.body?.message ?? `Request failed with status ${res.status}`,
      );
      return;
    }
    setIfCurrent(options.setResult, res.body);
  } catch (err) {
    if (isAbortError(err)) return;
    setIfCurrent(
      options.setError,
      err instanceof Error ? err.message : String(err),
    );
  } finally {
    if (timer) clearInterval(timer);
    setIfCurrent(options.setLoading, false);
  }
}

function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === "AbortError";
}

function supportsCodex(
  connector: ConnectorKind,
): connector is "github" | "jira" | "slack" | "googledrive" | "notion" | "sharepoint" {
  return (
    connector === "github" ||
    connector === "jira" ||
    connector === "slack" ||
    connector === "googledrive" ||
    connector === "notion" ||
    connector === "sharepoint"
  );
}

function cleanMetadata(
  metadata: Record<string, string> | undefined,
): Record<string, string> | undefined {
  if (!metadata) return undefined;
  const entries = Object.entries(metadata).filter(
    ([, value]) => value.trim() !== "",
  );
  if (entries.length === 0) return undefined;
  return Object.fromEntries(entries);
}
