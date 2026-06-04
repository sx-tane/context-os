import { streamCodexReauth } from "$lib/api";

interface ReauthRunnerOptions {
  plugin: string;
  refreshCodexStatus: () => Promise<void>;
  setPlugin: (plugin: string) => void;
  setLog: (log: string | ((current: string) => string)) => void;
  setRunning: (running: boolean) => void;
  isCurrent?: () => boolean;
  signal?: AbortSignal;
}

export async function runCodexReauth(options: ReauthRunnerOptions): Promise<void> {
  const setIfCurrent = <T>(setter: (value: T) => void, value: T) => {
    if (options.isCurrent?.() === false) return;
    setter(value);
  };

  setIfCurrent(options.setPlugin, options.plugin);
  setIfCurrent(options.setLog, "");
  setIfCurrent(options.setRunning, true);

  try {
    await streamCodexReauth(
      options.plugin,
      (line) => setIfCurrent(options.setLog, (current) => current + line + "\n"),
      { signal: options.signal },
    );
  } catch (err) {
    if (!isAbortError(err)) {
      setIfCurrent(options.setLog, (current) => current + String(err) + "\n");
    }
  } finally {
    setIfCurrent(options.setRunning, false);
    setIfCurrent(options.setPlugin, "");
    if (options.isCurrent?.() !== false) {
      await options.refreshCodexStatus();
    }
  }
}

function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === "AbortError";
}
