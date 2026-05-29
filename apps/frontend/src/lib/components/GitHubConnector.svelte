<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { IngestProvider, IngestResult, CodexPlugin } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import { runCodexReauth } from "$lib/reauthRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import CodexBadge from "./CodexBadge.svelte";
  import ResultPanel from "./IngestResult.svelte";

  // Shared Codex state from parent page
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  export let refreshCodexStatus: () => Promise<void>;

  // Local state
  let uri = "https://github.com/sx-tane/context-os/issues/1";
  let token = "";
  let provider: IngestProvider = "token";
  let loading = false;
  let errorMessage = "";
  let result: IngestResult | null = null;
  let liveLog = "";
  let elapsed = 0;
  let ingestController: AbortController | null = null;
  let reauthController: AbortController | null = null;
  let ingestRunID = 0;
  let reauthRunID = 0;

  let connected = false;
  let login = "";
  let name = "";

  // Re-auth state (local — only relevant to this connector's plugin)
  let reauthPlugin = "";
  let reauthLog = "";
  let reauthRunning = false;

  onMount(checkStatus);
  onDestroy(() => {
    ingestController?.abort();
    reauthController?.abort();
  });

  async function checkStatus() {
    const body = await getJSON<{ connected?: boolean; login?: string; name?: string }>("/github/status");
    connected = body?.connected === true;
    login = body?.login ?? "";
    name = body?.name ?? "";
  }

  async function runReauth(plugin: string) {
    reauthController?.abort();
    reauthController = new AbortController();
    const runID = ++reauthRunID;
    await runCodexReauth({
      plugin,
      refreshCodexStatus,
      signal: reauthController.signal,
      isCurrent: () => runID === reauthRunID,
      setPlugin: (value) => (reauthPlugin = value),
      setRunning: (value) => (reauthRunning = value),
      setLog: (value) => (reauthLog = typeof value === "function" ? value(reauthLog) : value),
    });
  }

  async function runIngest() {
    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    await runConnectorIngest({
      connector: "github",
      uri,
      token,
      provider,
      signal: ingestController.signal,
      isCurrent: () => runID === ingestRunID,
      setLoading: (value) => (loading = value),
      setError: (message) => (errorMessage = message),
      setResult: (value) => (result = value),
      setLiveLog: (value) => (liveLog = typeof value === "function" ? value(liveLog) : value),
      setElapsed: (value) => (elapsed = typeof value === "function" ? value(elapsed) : value),
    });
  }
</script>

<ConnectorCard
  title="GitHub MCP Connector"
  description="Ingest a GitHub repository, issue, or pull request via the MCP source connector."
  examples={["https://github.com/owner/repo", "https://github.com/owner/repo/issues/1", "repo://owner/repo/..."]}
>
  {#if connected}
    <div class="connector-badge">
      &#10003; Connected as <strong>{login}{name ? ` (${name})` : ""}</strong> via <code class="connector-card-code">Github</code>
    </div>
  {/if}

  <div class="connector-mode-toggle" aria-label="GitHub ingestion provider">
    <button type="button" class:active={provider === "token"} on:click={() => (provider = "token")}>Token / env</button>
    <button type="button" class:active={provider === "codex"} on:click={() => (provider = "codex")}>Codex CLI plugin</button>
  </div>

  <label class="connector-field">
    <span>URI</span>
    <input class="connector-input" type="text" bind:value={uri} placeholder="https://github.com/owner/repo/issues/1" />
  </label>

  {#if provider === "token"}
    <label class="connector-field">
      <span>GitHub token <span class="connector-optional">(optional — needed for private repos or if rate-limited)</span></span>
      <input class="connector-input" type="password" bind:value={token} placeholder="ghp_..." />
    </label>
    <details class="connector-help">
      <summary>How to get a GitHub token</summary>
      <ol>
        <li>Go to <a href="https://github.com/settings/tokens/new?scopes=repo&description=ContextOS" target="_blank" rel="noopener">github.com/settings/tokens</a></li>
        <li>Click <strong>Generate new token (classic)</strong></li>
        <li>Tick the <strong>repo</strong> scope</li>
        <li>Click <strong>Generate token</strong>, copy it, paste above</li>
      </ol>
      <p class="connector-note">
        Inside a Codespace you can leave this blank — <code class="connector-card-code">GITHUB_TOKEN</code> is already set to your account automatically.
      </p>
    </details>
  {:else}
    <CodexBadge
      {codexLoggedIn}
      {codexAccount}
      {codexPlugins}
      pluginName="github@openai-curated"
      {reauthRunning}
      {reauthPlugin}
      {reauthLog}
      on:reauth={(e) => runReauth(e.detail)}
    />
  {/if}

  <button class="connector-button" type="button" on:click={runIngest} disabled={loading || !uri.trim()}>
    {loading ? `Ingesting… (${elapsed}s)` : "Run ingest"}
  </button>

  {#if provider === "codex" && (liveLog || loading)}
    <pre class="connector-log">{liveLog || "Waiting for Codex output…"}</pre>
  {/if}

  {#if errorMessage}
    <div class="connector-error">{errorMessage}</div>
  {/if}

  {#if result}
    <ResultPanel {result} {provider} />
  {/if}
</ConnectorCard>
