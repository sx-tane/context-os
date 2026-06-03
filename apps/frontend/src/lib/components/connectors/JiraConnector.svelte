<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { CodexPlugin, IngestProvider, IngestResult } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { project } from "$lib/projectStore";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import CodexBadge from "./CodexBadge.svelte";
  import ResultPanel from "../feedback/IngestResult.svelte";
  import Button from "../ui/Button.svelte";
  import FormField from "../ui/FormField.svelte";
  import ModeToggle from "../ui/ModeToggle.svelte";
  import LogPanel from "../feedback/LogPanel.svelte";
  import ErrorPanel from "../feedback/ErrorPanel.svelte";

  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  export let refreshCodexStatus: () => Promise<void>;

  let uri = "";
  let token = "";
  let email = "";
  let apiBaseURL = "";
  let provider: IngestProvider = "codex";
  let loading = false;
  let errorMessage = "";
  let result: IngestResult | null = null;
  let liveLog = "";
  let elapsed = 0;
  let ingestController: AbortController | null = null;
  let ingestRunID = 0;

  let connected = false;
  let baseURL = "";
  let tokenConfigured = false;
  let emailConfigured = false;

  onMount(checkStatus);
  onDestroy(() => {
    ingestController?.abort();
  });

  async function checkStatus() {
    const body = await getJSON<{
      connected?: boolean;
      base_url?: string;
      token_configured?: boolean;
      email_configured?: boolean;
    }>("/jira/status");
    connected = body?.connected === true;
    baseURL = body?.base_url ?? "";
    tokenConfigured = body?.token_configured === true;
    emailConfigured = body?.email_configured === true;
  }

  async function runIngest() {
    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    await runConnectorIngest({
      connector: "jira",
      workspace_id: $project.workspacePath,
      uri,
      token,
      provider,
      metadata: {
        jira_email: email,
        jira_api_base_url: apiBaseURL,
      },
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
  title="Jira/Rovo MCP Connector"
  description="Ingest Jira issue or project context through direct API auth or the Atlassian Rovo Codex plugin."
  examples={[
    "https://site.atlassian.net/browse/PROJ-123",
    "jira://issue/PROJ-123",
    "jira://project/PROJ",
  ]}
>
  <ModeToggle
    bind:value={provider}
    options={[
      { value: "token", label: "Token / env" },
      { value: "codex", label: "Codex Rovo plugin" },
    ]}
    ariaLabel="Jira ingestion provider"
  />

  {#if provider === "token"}
    {#if connected}
      <div class="connector-badge">
        &#10003; Jira base URL configured{baseURL
          ? `: ${baseURL}`
          : ""}{tokenConfigured ? " with token" : ""}{emailConfigured
          ? " and email"
          : ""}
      </div>
    {/if}
    <FormField
      label="Jira API token"
      optional="(optional when env token is set)"
      type="password"
      bind:value={token}
      placeholder="ATATT..."
    />
    <FormField
      label="Jira email"
      optional="(optional when env email is set)"
      bind:value={email}
      placeholder="name@example.com"
    />
    <FormField
      label="Jira base URL"
      optional="(optional when env base URL is set)"
      bind:value={apiBaseURL}
      placeholder="https://site.atlassian.net"
    />
  {:else}
    <CodexBadge
      {codexLoggedIn}
      {codexAccount}
      {codexPlugins}
      pluginName="atlassian-rovo@openai-curated"
    />
  {/if}

  <FormField
    label="URI"
    bind:value={uri}
    placeholder="https://site.atlassian.net/browse/PROJ-123"
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
