<script lang="ts">
  import { onMount } from "svelte";

  const API_URL = "/api";
  const WORKER_URL = "/worker";

  type ServiceStatus = "checking" | "ok" | "unreachable";

  let apiStatus: ServiceStatus = "checking";
  let workerStatus: ServiceStatus = "checking";

  async function probe(url: string): Promise<ServiceStatus> {
    try {
      const res = await fetch(`${url}/health`, {
        signal: AbortSignal.timeout(3000),
      });
      return res.ok ? "ok" : "unreachable";
    } catch {
      return "unreachable";
    }
  }

  onMount(async () => {
    [apiStatus, workerStatus] = await Promise.all([
      probe(API_URL),
      probe(WORKER_URL),
    ]);
    await Promise.all([checkGithubStatus(), checkSlackStatus()]);
  });

  const label: Record<ServiceStatus, string> = {
    checking: "Checking...",
    ok: "Online",
    unreachable: "Offline",
  };

  const color: Record<ServiceStatus, string> = {
    checking: "#888",
    ok: "#22c55e",
    unreachable: "#ef4444",
  };

  // GitHub MCP connector tester ----------------------------------------------
  type IngestProvider = "token" | "codex";

  let uri = "https://github.com/sx-tane/context-os/issues/1";
  let token = "";
  let githubProvider: IngestProvider = "token";
  let loading = false;
  let errorMessage = "";
  let githubConnected = false;
  let githubLogin = "";
  let githubName = "";
  let result: {
    connector: string;
    capabilities: string[];
    event: {
      id: string;
      type: string;
      source: string;
      source_id: string;
      subject: string;
      occurred_at: string;
    };
    preview: string;
    metadata: Record<string, string>;
  } | null = null;

  async function checkGithubStatus() {
    try {
      const res = await fetch(`${API_URL}/github/status`);
      if (res.ok) {
        const body = await res.json();
        githubConnected = body?.connected === true;
        githubLogin = body?.login ?? "";
        githubName = body?.name ?? "";
      }
    } catch {
      // ignore — connector falls back gracefully
    }
  }

  async function runIngest() {
    loading = true;
    errorMessage = "";
    result = null;
    try {
      const res = await fetch(`${API_URL}/github/ingest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          uri,
          token: token || undefined,
          provider: githubProvider,
        }),
      });
      const body = await res.json();
      if (!res.ok) {
        errorMessage =
          body?.message ?? `Request failed with status ${res.status}`;
        return;
      }
      result = body;
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  // Slack MCP connector tester -----------------------------------------------
  let slackURI = "slack://C1234567890";
  let slackToken = "";
  let slackProvider: IngestProvider = "token";
  let slackLoading = false;
  let slackError = "";
  let slackConnected = false;
  let slackSource = "none"; // "env" | "oauth" | "none"
  let slackTeamName = "";
  let slackResult: {
    connector: string;
    capabilities: string[];
    event: {
      id: string;
      type: string;
      source: string;
      source_id: string;
      subject: string;
      occurred_at: string;
    };
    preview: string;
    metadata: Record<string, string>;
  } | null = null;

  async function checkSlackStatus() {
    try {
      const res = await fetch(`${API_URL}/slack/status`);
      if (res.ok) {
        const body = await res.json();
        slackConnected = body?.connected === true;
        slackSource = body?.source ?? "none";
        slackTeamName = body?.team_name ?? "";
      }
    } catch {
      // ignore — connector falls back gracefully
    }
  }

  async function runSlackIngest() {
    slackLoading = true;
    slackError = "";
    slackResult = null;
    try {
      const res = await fetch(`${API_URL}/slack/ingest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          uri: slackURI,
          token: slackToken || undefined,
          provider: slackProvider,
        }),
      });
      const body = await res.json();
      if (!res.ok) {
        slackError =
          body?.message ?? `Request failed with status ${res.status}`;
        return;
      }
      slackResult = body;
    } catch (err) {
      slackError = err instanceof Error ? err.message : String(err);
    } finally {
      slackLoading = false;
    }
  }
</script>

<svelte:head>
  <title>ContextOS</title>
</svelte:head>

<main>
  <h1>ContextOS</h1>

  <section class="status">
    <h2>System Status</h2>
    <div class="row">
      <span class="dot" style="background:{color[apiStatus]}"></span>
      <span class="service">API</span>
      <span class="value" style="color:{color[apiStatus]}"
        >{label[apiStatus]}</span
      >
    </div>
    <div class="row">
      <span class="dot" style="background:{color[workerStatus]}"></span>
      <span class="service">AI Worker</span>
      <span class="value" style="color:{color[workerStatus]}"
        >{label[workerStatus]}</span
      >
    </div>
  </section>

  <section class="card">
    <h2>GitHub MCP Connector</h2>
    <p class="hint">
      Ingest a GitHub repository, issue, or pull request via the MCP source
      connector. Accepts <code>https://github.com/owner/repo</code>,
      <code>.../issues/N</code>, <code>.../pull/N</code>, or
      <code>repo://owner/repo/...</code> URIs.
    </p>

    {#if githubConnected}
      <div class="connected-badge">
        &#10003; Connected as
        <strong>{githubLogin}{githubName ? ` (${githubName})` : ""}</strong>
        via <code>GITHUB_TOKEN</code>
      </div>
    {/if}

    <div class="mode-toggle" aria-label="GitHub ingestion provider">
      <button
        type="button"
        class:active={githubProvider === "token"}
        on:click={() => (githubProvider = "token")}>Token / env</button
      >
      <button
        type="button"
        class:active={githubProvider === "codex"}
        on:click={() => (githubProvider = "codex")}>Codex CLI plugin</button
      >
    </div>

    <label>
      <span>URI</span>
      <input
        type="text"
        bind:value={uri}
        placeholder="https://github.com/owner/repo/issues/1"
      />
    </label>

    {#if githubProvider === "token"}
      <label>
        <span
          >GitHub token <span class="optional"
            >(optional — needed for private repos or if rate-limited)</span
          ></span
        >
        <input type="password" bind:value={token} placeholder="ghp_..." />
      </label>
      <details class="token-help">
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
        <p class="token-note">
          Inside a Codespace you can leave this blank — <code>GITHUB_TOKEN</code
          > is already set to your account automatically.
        </p>
      </details>
    {:else}
      <div class="plugin-note">
        Codex account required — run <code>codex login</code> once.
      </div>
      <label>
        <span
          >GitHub token <span class="optional"
            >(optional — overrides the logged-in Codex account; lets you use a
            different GitHub account per request)</span
          ></span
        >
        <input type="password" bind:value={token} placeholder="ghp_..." />
      </label>
    {/if}

    <button on:click={runIngest} disabled={loading || !uri.trim()}>
      {loading ? "Ingesting..." : "Run ingest"}
    </button>

    {#if errorMessage}
      <div class="error">{errorMessage}</div>
    {/if}

    {#if result}
      <div class="result">
        <div class="kv">
          <strong>Connector</strong><span>{result.connector}</span>
        </div>
        <div class="kv">
          <strong>Capabilities</strong><span
            >{result.capabilities.join(", ")}</span
          >
        </div>
        <div class="kv">
          <strong>Event ID</strong><span>{result.event.id}</span>
        </div>
        <div class="kv">
          <strong>Event type</strong><span>{result.event.type}</span>
        </div>
        <div class="kv">
          <strong>Source ID</strong><span>{result.event.source_id}</span>
        </div>
        <div class="kv">
          <strong>Subject</strong><span>{result.event.subject}</span>
        </div>
        <div class="kv">
          <strong>Occurred at</strong><span>{result.event.occurred_at}</span>
        </div>

        <details open>
          <summary>Metadata</summary>
          <pre>{JSON.stringify(result.metadata, null, 2)}</pre>
        </details>

        {#if githubProvider === "codex" && result.metadata?.codex_log}
          <details open>
            <summary>Codex log</summary>
            <pre>{result.metadata.codex_log}</pre>
          </details>
        {/if}

        <details>
          <summary>Content</summary>
          <pre>{(() => {
              try {
                return JSON.stringify(JSON.parse(result.preview), null, 2);
              } catch {
                return result.preview;
              }
            })()}</pre>
        </details>
      </div>
    {/if}
  </section>

  <section class="card">
    <h2>Slack MCP Connector</h2>
    <p class="hint">
      Ingest a Slack channel or message. Use <code>slack://CHANNEL_ID</code> for
      a channel or <code>slack://CHANNEL_ID/TIMESTAMP</code> for a message.
    </p>

    <div class="mode-toggle" aria-label="Slack ingestion provider">
      <button
        type="button"
        class:active={slackProvider === "token"}
        on:click={() => (slackProvider = "token")}>Token / env</button
      >
      <button
        type="button"
        class:active={slackProvider === "codex"}
        on:click={() => (slackProvider = "codex")}>Codex CLI plugin</button
      >
    </div>

    {#if slackProvider === "token"}
      {#if slackConnected && slackSource === "oauth"}
        <div class="connected-badge">
          &#10003; Connected to <strong>{slackTeamName}</strong> via saved token
        </div>
      {:else if slackConnected}
        <div class="connected-badge">
          &#10003; Connected via <code>SLACK_BOT_TOKEN</code>
        </div>
      {/if}
      <details class="token-help">
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
            Under <strong>OAuth &amp; Permissions</strong>, add Bot Token
            Scopes:
            <code>channels:history</code>,
            <code>channels:read</code>
          </li>
          <li>Install the app to your workspace</li>
          <li>Copy the Bot User OAuth Token and paste it below</li>
        </ol>
        <p class="token-note">
          You can also set <code>SLACK_BOT_TOKEN</code> before starting the API.
        </p>
      </details>
      <label>
        <span
          >Slack token <span class="optional"
            >(optional when env token is set)</span
          ></span
        >
        <input type="password" bind:value={slackToken} placeholder="xoxb-..." />
      </label>
    {:else}
      <div class="plugin-note">
        Codex account required — run <code>codex login</code> once.
      </div>
      <label>
        <span
          >Slack bot token <span class="optional"
            >(optional — overrides the logged-in Codex account; lets you use a
            different Slack workspace per request)</span
          ></span
        >
        <input type="password" bind:value={slackToken} placeholder="xoxb-..." />
      </label>
    {/if}

    <label style="margin-top:0.75rem">
      <span>URI</span>
      <input
        type="text"
        bind:value={slackURI}
        placeholder="slack://C1234567890"
      />
    </label>

    <button
      on:click={runSlackIngest}
      disabled={slackLoading || !slackURI.trim()}
    >
      {slackLoading ? "Ingesting..." : "Run ingest"}
    </button>

    {#if slackError}
      <div class="error">{slackError}</div>
    {/if}

    {#if slackResult}
      <div class="result">
        <div class="kv">
          <strong>Connector</strong><span>{slackResult.connector}</span>
        </div>
        <div class="kv">
          <strong>Capabilities</strong><span
            >{slackResult.capabilities.join(", ")}</span
          >
        </div>
        <div class="kv">
          <strong>Event ID</strong><span>{slackResult.event.id}</span>
        </div>
        <div class="kv">
          <strong>Event type</strong><span>{slackResult.event.type}</span>
        </div>
        <div class="kv">
          <strong>Source ID</strong><span>{slackResult.event.source_id}</span>
        </div>
        <div class="kv">
          <strong>Subject</strong><span>{slackResult.event.subject}</span>
        </div>
        <div class="kv">
          <strong>Occurred at</strong><span
            >{slackResult.event.occurred_at}</span
          >
        </div>

        <details open>
          <summary>Metadata</summary>
          <pre>{JSON.stringify(slackResult.metadata, null, 2)}</pre>
        </details>

        {#if slackProvider === "codex" && slackResult.metadata?.codex_log}
          <details open>
            <summary>Codex log</summary>
            <pre>{slackResult.metadata.codex_log}</pre>
          </details>
        {/if}

        <details>
          <summary>Content</summary>
          <pre>{(() => {
              try {
                return JSON.stringify(JSON.parse(slackResult.preview), null, 2);
              } catch {
                return slackResult.preview;
              }
            })()}</pre>
        </details>
      </div>
    {/if}
  </section>
</main>

<style>
  main {
    font-family: system-ui, sans-serif;
    max-width: 720px;
    margin: 3rem auto;
    padding: 0 1rem;
  }

  h1 {
    font-size: 1.75rem;
    margin-bottom: 1.5rem;
  }

  .status,
  .card {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 1.25rem 1.5rem;
    margin-bottom: 1.5rem;
  }

  h2 {
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #6b7280;
    margin: 0 0 1rem;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .service {
    flex: 1;
    font-weight: 500;
  }

  .value {
    font-weight: 600;
  }

  .hint {
    color: #6b7280;
    font-size: 0.85rem;
    margin: 0 0 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.75rem;
    font-size: 0.85rem;
  }

  label > span {
    display: block;
    margin-bottom: 0.25rem;
    color: #374151;
  }

  .optional {
    font-weight: 400;
    color: #9ca3af;
    font-size: 0.8rem;
  }

  .token-help {
    margin: -0.25rem 0 0.75rem;
    font-size: 0.8rem;
    color: #6b7280;
  }

  .token-help summary {
    cursor: pointer;
    color: #2563eb;
    user-select: none;
  }

  .token-help ol {
    margin: 0.5rem 0 0.5rem 1.25rem;
    padding: 0;
    line-height: 1.8;
  }

  .token-note {
    margin: 0.25rem 0 0;
    color: #9ca3af;
  }

  .mode-toggle {
    display: inline-flex;
    gap: 0.25rem;
    padding: 0.25rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    margin: 0 0 0.75rem;
    background: #f9fafb;
  }

  .mode-toggle button {
    background: transparent;
    color: #374151;
    border: 0;
    padding: 0.4rem 0.65rem;
    border-radius: 4px;
  }

  .mode-toggle button.active {
    background: #111827;
    color: white;
  }

  .plugin-note {
    margin: 0 0 0.75rem;
    padding: 0.75rem 0.9rem;
    background: #eff6ff;
    color: #1e3a8a;
    border: 1px solid #bfdbfe;
    border-radius: 6px;
    font-size: 0.85rem;
    line-height: 1.5;
  }

  input {
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font: inherit;
    box-sizing: border-box;
  }

  button {
    background: #111827;
    color: white;
    border: 0;
    padding: 0.55rem 1rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .error {
    margin-top: 1rem;
    padding: 0.75rem 1rem;
    background: #fef2f2;
    color: #991b1b;
    border: 1px solid #fecaca;
    border-radius: 6px;
    font-size: 0.85rem;
    white-space: pre-wrap;
  }

  .result {
    margin-top: 1rem;
    border-top: 1px solid #e5e7eb;
    padding-top: 1rem;
  }

  .kv {
    display: flex;
    gap: 0.75rem;
    padding: 0.25rem 0;
    font-size: 0.85rem;
    word-break: break-all;
  }

  .kv strong {
    color: #374151;
    min-width: 8rem;
  }

  details {
    margin-top: 0.75rem;
    font-size: 0.85rem;
  }

  pre {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    padding: 0.75rem;
    border-radius: 6px;
    max-height: 320px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }

  code {
    background: #f3f4f6;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
  }

  .connected-badge {
    display: inline-block;
    margin-bottom: 0.75rem;
    padding: 0.3rem 0.75rem;
    background: #f0fdf4;
    color: #166534;
    border: 1px solid #bbf7d0;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
  }
</style>
