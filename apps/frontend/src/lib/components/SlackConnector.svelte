<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { IngestProvider, IngestResult, CodexPlugin } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import { runCodexReauth } from "$lib/reauthRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import CodexBadge from "./CodexBadge.svelte";
  import ResultPanel from "./IngestResult.svelte";
  import Button from "./Button.svelte";
  import FormField from "./FormField.svelte";
  import ModeToggle from "./ModeToggle.svelte";
  import LogPanel from "./LogPanel.svelte";
  import ErrorPanel from "./ErrorPanel.svelte";

  // Shared Codex state from parent page
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  export let refreshCodexStatus: () => Promise<void>;

  // Local state
  let uri = "slack://C1234567890";
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
  let source = "none";
  let teamName = "";
  let userName = "";

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
      source?: string;
      team_name?: string;
      user_name?: string;
    }>("/slack/status");
    connected = body?.connected === true;
    source = body?.source ?? "none";
    teamName = body?.team_name ?? "";
    userName = body?.user_name ?? "";
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
      connector: "slack",
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
  title="Slack MCP Connector"
  description="Ingest a Slack channel or message."
  examples={["slack://CHANNEL_ID", "slack://CHANNEL_ID/TIMESTAMP"]}
>
  <ModeToggle
    bind:value={provider}
    options={[
      { value: "token", label: "Token / env" },
      { value: "codex", label: "Codex CLI plugin" },
    ]}
    ariaLabel="Slack ingestion provider"
  />

  {#if provider === "token"}
    {#if connected && source === "oauth"}
      <div class="connector-badge">
        &#10003; Connected to <strong>{teamName}</strong> via saved token
      </div>
    {:else if connected}
      <div class="connector-badge">
        &#10003; Connected via <code class="connector-card-code"
          >SLACK_BOT_TOKEN</code
        >
      </div>
    {/if}
    <details class="connector-help">
      <summary>How to get a Slack bot token</summary>
      <ol>
        <li>
          Go to <a
            href="https://api.slack.com/apps"
            target="_blank"
            rel="noopener">api.slack.com/apps</a
          >
          → <strong>Create New App → From scratch</strong>
        </li>
        <li>
          Under <strong>OAuth &amp; Permissions</strong>, add Bot Token Scopes:
          <code>channels:history</code>, <code>channels:read</code>
        </li>
        <li>Install the app to your workspace</li>
        <li>Copy the Bot User OAuth Token and paste it below</li>
      </ol>
      <p class="connector-note">
        You can also set <code class="connector-card-code">SLACK_BOT_TOKEN</code
        > before starting the API.
      </p>
    </details>
    <FormField
      label="Slack token"
      optional="(optional when env token is set)"
      type="password"
      bind:value={token}
      placeholder="xoxb-..."
    />
  {:else}
    <CodexBadge
      {codexLoggedIn}
      {codexAccount}
      {codexPlugins}
      pluginName="slack@openai-curated"
      {reauthRunning}
      {reauthPlugin}
      {reauthLog}
      on:reauth={(e) => runReauth(e.detail)}
    />
  {/if}

  <FormField
    label="URI"
    bind:value={uri}
    placeholder="slack://C1234567890"
    offset
  />

  <Button {loading} disabled={loading || !uri.trim()} on:click={runIngest}>
    {loading ? `Ingesting… (${elapsed}s)` : "Run ingest"}
  </Button>

  <LogPanel log={liveLog} {loading} visible={provider === "codex"} />

  <ErrorPanel message={errorMessage} />

  {#if result}
    <ResultPanel {result} {provider} />
  {/if}
</ConnectorCard>
