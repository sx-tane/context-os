<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { IngestProvider, IngestResult, CodexPlugin } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import { runCodexReauth } from "$lib/reauthRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import CodexBadge from "./CodexBadge.svelte";
  import ResultPanel from "../feedback/IngestResult.svelte";
  import Button from "../ui/Button.svelte";
  import FormField from "../ui/FormField.svelte";
  import ModeToggle from "../ui/ModeToggle.svelte";
  import LogPanel from "../feedback/LogPanel.svelte";
  import ErrorPanel from "../feedback/ErrorPanel.svelte";

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
    const body = await getJSON<{
      connected?: boolean;
      login?: string;
      name?: string;
    }>("/github/status");
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
      setLog: (value) =>
        (reauthLog = typeof value === "function" ? value(reauthLog) : value),
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
      setLiveLog: (value) =>
        (liveLog = typeof value === "function" ? value(liveLog) : value),
      setElapsed: (value) =>
        (elapsed = typeof value === "function" ? value(elapsed) : value),
    });
  }
</script>

<ConnectorCard
  title="GitHub MCP Connector"
  description="Ingest a GitHub repository, issue, or pull request via the MCP source connector."
  examples={[
    "https://github.com/owner/repo",
    "https://github.com/owner/repo/issues/1",
    "repo://owner/repo/...",
  ]}
>
  {#if connected}
    <div class="connector-badge">
      &#10003; Connected as <strong>{login}{name ? ` (${name})` : ""}</strong>
      via <code class="connector-card-code">Github</code>
    </div>
  {/if}

  <ModeToggle
    bind:value={provider}
    options={[
      { value: "token", label: "Token / env" },
      { value: "codex", label: "Codex CLI plugin" },
    ]}
    ariaLabel="GitHub ingestion provider"
  />

  <FormField
    label="URI"
    bind:value={uri}
    placeholder="https://github.com/owner/repo/issues/1"
  />

  {#if provider === "token"}
    <FormField
      label="GitHub token"
      optional="(optional — needed for private repos or if rate-limited)"
      type="password"
      bind:value={token}
      placeholder="ghp_..."
    />
    <details class="connector-help">
      <summary>How to get a GitHub token</summary>
      <ol>
        <li>
          Go to <a
            href="https://github.com/settings/tokens/new?scopes=repo&description=ContextOS"
            target="_blank"
            rel="noopener">github.com/settings/tokens</a
          >
        </li>
        <li>Click <strong>Generate new token (classic)</strong></li>
        <li>Tick the <strong>repo</strong> scope</li>
        <li>Click <strong>Generate token</strong>, copy it, paste above</li>
      </ol>
      <p class="connector-note">
        Inside a Codespace you can leave this blank — <code
          class="connector-card-code">GITHUB_TOKEN</code
        > is already set to your account automatically.
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

  <Button {loading} disabled={loading || !uri.trim()} on:click={runIngest}>
    {loading ? `Ingesting\u2026 (${elapsed}s)` : "Run ingest"}
  </Button>

  <LogPanel log={liveLog} {loading} visible={provider === "codex"} />

  <ErrorPanel message={errorMessage} />

  {#if result}
    <ResultPanel {result} {provider} />
  {/if}
</ConnectorCard>
