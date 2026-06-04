# Frontend Connector Skeleton

Copy this file to `apps/frontend/src/lib/components/connectors/<Name>Connector.svelte`.
Replace all `<Name>` / `<name>` / `<scheme>` / `<EnvField>` placeholders.
Delete any block marked `<!-- only if ... -->` that does not apply to your connector.

---

```svelte
<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { IngestProvider, IngestResult, CodexPlugin } from "$lib/types";
  import { getJSON } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import CodexBadge from "./CodexBadge.svelte";          <!-- only if connector has a Codex plugin -->
  import Button from "../ui/Button.svelte";
  import FormField from "../ui/FormField.svelte";
  import ModeToggle from "../ui/ModeToggle.svelte";      <!-- only if provider switching is needed -->
  import LogPanel from "../feedback/LogPanel.svelte";
  import ErrorPanel from "../feedback/ErrorPanel.svelte";
  import ResultPanel from "../feedback/IngestResult.svelte";

  // Shared Codex state from parent page — always present, always these 4 props
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  export let refreshCodexStatus: () => Promise<void>;

  // Local state
  let uri = "<scheme>://<example-resource>";
  let token = "";
  let provider: IngestProvider = "token";   // change default to "codex" if applicable
  let loading = false;
  let errorMessage = "";
  let result: IngestResult | null = null;
  let liveLog = "";
  let elapsed = 0;
  let ingestController: AbortController | null = null;
  let ingestRunID = 0;

  // Status state
  let connected = false;
  // Add more connector-specific status fields here, e.g.:
  // let teamName = "";

  onMount(checkStatus);
  onDestroy(() => {
    ingestController?.abort();
  });

  async function checkStatus() {
    const body = await getJSON<{
      connected?: boolean;
      // Add connector-specific status fields here, e.g.:
      // team_name?: string;
    }>("/<name>/status");
    connected = body?.connected === true;
    // Map additional fields, e.g.:
    // teamName = body?.team_name ?? "";
  }

  // runReauth is not currently wired in the UI.
  // To reconnect a plugin to a different account, run in terminal:
  // codex plugin remove <plugin>@openai-curated && codex plugin add <plugin>@openai-curated

  async function runIngest() {
    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    await runConnectorIngest({
      connector: "<name>",
      uri,
      token,
      provider,
      signal: ingestController.signal,
      isCurrent: () => runID === ingestRunID,
      setLoading: (value) => (loading       = value),
      setError:   (message) => (errorMessage = message),
      setResult:  (value) => (result         = value),
      setLiveLog: (value) => (liveLog  = typeof value === "function" ? value(liveLog)  : value),
      setElapsed: (value) => (elapsed  = typeof value === "function" ? value(elapsed)  : value),
    });
  }
</script>

<ConnectorCard
  title="<Name> Connector"
  description="Ingest a <Name> <artifact description> via the MCP source connector."
  examples={[
    "<scheme>://<example-1>",
    "<scheme>://<example-2>",
  ]}
>
  {#if connected}
    <div class="connector-badge">
      &#10003; Connected
      <!-- Optionally show identity: as <strong>{teamName}</strong> -->
    </div>
  {/if}

  <!-- only if provider switching is needed -->
  <ModeToggle
    bind:value={provider}
    options={[
      { value: "token", label: "Token / env" },
      { value: "codex", label: "Codex CLI plugin" },
    ]}
    ariaLabel="<Name> ingestion provider"
  />

  <FormField
    label="URI"
    bind:value={uri}
    placeholder="<scheme>://<example-resource>"
  />

  {#if provider === "token"}
    <FormField
      label="<Name> token"
      optional="(optional — falls back to <EnvVar> env var)"
      type="password"
      bind:value={token}
      placeholder="<token-prefix>..."
    />
  {:else}
    <!-- only if connector has a Codex plugin -->
    <CodexBadge
      {codexLoggedIn}
      {codexAccount}
      {codexPlugins}
      pluginName="<codex-plugin-name>"
    />
  {/if}

  <Button on:click={runIngest} disabled={loading}>
    {loading ? `Ingesting… (${elapsed}s)` : "Ingest"}
  </Button>

  {#if liveLog}
    <LogPanel log={liveLog} />
  {/if}

  {#if errorMessage}
    <ErrorPanel message={errorMessage} />
  {/if}

  {#if result}
    <ResultPanel {result} />
  {/if}
</ConnectorCard>
```

---

## Registration in `+page.svelte`

### 1. Add import at the top of the `<script>` block

```ts
import <Name>Connector from "$lib/components/connectors/<Name>Connector.svelte";
```

### 2. Add the component in the `<main>` template

```svelte
<<Name>Connector
  {codexLoggedIn}
  {codexAccount}
  {codexPlugins}
  {refreshCodexStatus}
/>
```

Place it after the last existing connector and before any closing tags.
