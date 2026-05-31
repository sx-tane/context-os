<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { CodexPlugin, IngestResult } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import Button from "../ui/Button.svelte";
  import FormField from "../ui/FormField.svelte";
  import ErrorPanel from "../feedback/ErrorPanel.svelte";
  import ResultPanel from "../feedback/IngestResult.svelte";

  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  export let refreshCodexStatus: () => Promise<void>;

  // These four Codex props are part of the uniform connector interface.
  // Google Drive does not use Codex streaming but receives them for API consistency.
  void codexLoggedIn, codexAccount, codexPlugins, refreshCodexStatus;

  let uri = "";
  let credentialPath = "";
  let serviceAccountPath = "";
  let accessToken = "";
  let cursor = "";
  let loading = false;
  let errorMessage = "";
  let result: IngestResult | null = null;
  let elapsed = 0;
  let ingestController: AbortController | null = null;
  let ingestRunID = 0;

  let connected = false;
  let oauthConfigured = false;
  let serviceAccountConfigured = false;
  let accessTokenConfigured = false;
  let folderConfigured = false;

  onMount(checkStatus);
  onDestroy(() => {
    ingestController?.abort();
  });

  async function checkStatus() {
    const body = await getJSON<{
      connected?: boolean;
      oauth_configured?: boolean;
      service_account_configured?: boolean;
      access_token_configured?: boolean;
      folder_configured?: boolean;
    }>("/googledrive/status");
    connected = body?.connected === true;
    oauthConfigured = body?.oauth_configured === true;
    serviceAccountConfigured = body?.service_account_configured === true;
    accessTokenConfigured = body?.access_token_configured === true;
    folderConfigured = body?.folder_configured === true;
  }

  async function runIngest() {
    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    await runConnectorIngest({
      connector: "googledrive",
      uri,
      cursor: cursor || undefined,
      provider: "token",
      metadata: {
        googledrive_oauth_credentials_path: credentialPath,
        googledrive_service_account_path: serviceAccountPath,
        googledrive_access_token: accessToken,
      },
      signal: ingestController.signal,
      isCurrent: () => runID === ingestRunID,
      setLoading: (value) => (loading = value),
      setError: (message) => (errorMessage = message),
      setResult: (value) => (result = value),
      setLiveLog: (_value) => {},
      setElapsed: (value) =>
        (elapsed = typeof value === "function" ? value(elapsed) : value),
    });
  }
</script>

<ConnectorCard
  title="Google Drive MCP Connector"
  description="Ingest Google Docs, Sheets, and Slides from a Drive folder via OAuth or service-account credentials."
  examples={[
    "https://drive.google.com/drive/folders/1234567890",
    "googledrive://folder/1234567890",
  ]}
>
  {#if connected}
    <div class="connector-badge">
      &#10003; Google Drive credentials configured{oauthConfigured
        ? " (OAuth)"
        : ""}{serviceAccountConfigured ? " (service account)" : ""}{accessTokenConfigured
        ? " (access token)"
        : ""}{folderConfigured ? " · default folder set" : ""}
    </div>
  {/if}

  <FormField
    label="OAuth credentials path"
    optional="(optional when GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH is set)"
    bind:value={credentialPath}
    placeholder="/Users/name/.config/context-os/google-authorized-user.json"
  />

  <FormField
    label="Service account path"
    optional="(optional when GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH is set)"
    bind:value={serviceAccountPath}
    placeholder="/Users/name/.config/context-os/google-service-account.json"
  />

  <FormField
    label="Access token"
    optional="(optional override)"
    type="password"
    bind:value={accessToken}
    placeholder="ya29.a0AfH6SMD..."
  />

  <FormField
    label="Folder URI"
    bind:value={uri}
    placeholder="https://drive.google.com/drive/folders/1234567890"
    offset
  />

  <FormField
    label="Cursor"
    optional="(optional RFC3339 modified-time watermark)"
    bind:value={cursor}
    placeholder="2026-05-29T10:00:00Z"
  />

  <Button
    {loading}
    disabled={loading || (!uri.trim() && !folderConfigured)}
    on:click={runIngest}
  >
    {loading ? `Ingesting… (${elapsed}s)` : "Run ingest"}
  </Button>

  <ErrorPanel message={errorMessage} />

  {#if result}
    <ResultPanel {result} provider="token" />
  {/if}
</ConnectorCard>
