<script lang="ts">
  import { onMount } from "svelte";
  import type { IngestProvider, IngestResult, CodexPlugin } from "$lib/types";
  import CodexBadge from "./CodexBadge.svelte";
  import ResultPanel from "./IngestResult.svelte";

  const API_URL = "/api";

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
  let _timer: ReturnType<typeof setInterval> | null = null;

  let connected = false;
  let source = "none";
  let teamName = "";
  let userName = "";

  // Re-auth state (local — only relevant to this connector's plugin)
  let reauthPlugin = "";
  let reauthLog = "";
  let reauthRunning = false;

  onMount(checkStatus);

  async function checkStatus() {
    try {
      const res = await fetch(`${API_URL}/slack/status`);
      if (res.ok) {
        const body = await res.json();
        connected = body?.connected === true;
        source = body?.source ?? "none";
        teamName = body?.team_name ?? "";
        userName = body?.user_name ?? "";
      }
    } catch {
      // ignore — connector falls back gracefully
    }
  }

  async function runReauth(plugin: string) {
    reauthPlugin = plugin;
    reauthLog = "";
    reauthRunning = true;
    try {
      const res = await fetch(`${API_URL}/codex/plugin-reauth?plugin=${plugin}`, { method: "POST" });
      if (!res.body) throw new Error("No response body");
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buf = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        const blocks = buf.split("\n\n");
        buf = blocks.pop() ?? "";
        for (const block of blocks) {
          const dataLine = block.split("\n").find((l) => l.startsWith("data:"));
          if (dataLine) reauthLog += dataLine.slice(5).trim() + "\n";
        }
      }
    } catch (e) {
      reauthLog += String(e) + "\n";
    } finally {
      reauthRunning = false;
      reauthPlugin = "";
      await refreshCodexStatus();
    }
  }

  async function runIngest() {
    loading = true;
    errorMessage = "";
    result = null;
    liveLog = "";
    elapsed = 0;

    if (provider === "codex") {
      _timer = setInterval(() => { elapsed += 1; }, 1000);
      try {
        const res = await fetch(`${API_URL}/slack/ingest/stream`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ uri, token: token || undefined, provider: "codex" }),
        });
        if (!res.body) throw new Error("No response body");
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buf += decoder.decode(value, { stream: true });
          const blocks = buf.split("\n\n");
          buf = blocks.pop() ?? "";
          for (const block of blocks) {
            const eventLine = block.split("\n").find((l) => l.startsWith("event:"));
            const dataLine = block.split("\n").find((l) => l.startsWith("data:"));
            if (!eventLine || !dataLine) continue;
            const evType = eventLine.slice(6).trim();
            const data = dataLine.slice(5).trim();
            if (evType === "log") {
              liveLog += data + "\n";
            } else if (evType === "status") {
              const s = JSON.parse(data);
              if (s?.elapsed !== undefined) elapsed = s.elapsed;
            } else if (evType === "result") {
              result = JSON.parse(data);
            } else if (evType === "error") {
              const parsed = JSON.parse(data);
              errorMessage = parsed.message ?? data;
            }
          }
        }
      } catch (err) {
        errorMessage = err instanceof Error ? err.message : String(err);
      } finally {
        if (_timer) clearInterval(_timer);
        loading = false;
      }
      return;
    }

    try {
      const res = await fetch(`${API_URL}/slack/ingest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ uri, token: token || undefined, provider }),
      });
      const body = await res.json();
      if (!res.ok) {
        errorMessage = body?.message ?? `Request failed with status ${res.status}`;
        return;
      }
      result = body;
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }
</script>

<section class="card">
  <h2>Slack MCP Connector</h2>
  <p class="hint">
    Ingest a Slack channel or message. Use <code>slack://CHANNEL_ID</code> for a channel
    or <code>slack://CHANNEL_ID/TIMESTAMP</code> for a message.
  </p>

  <div class="mode-toggle" aria-label="Slack ingestion provider">
    <button type="button" class:active={provider === "token"} on:click={() => (provider = "token")}>Token / env</button>
    <button type="button" class:active={provider === "codex"} on:click={() => (provider = "codex")}>Codex CLI plugin</button>
  </div>

  {#if provider === "token"}
    {#if connected && source === "oauth"}
      <div class="connected-badge">&#10003; Connected to <strong>{teamName}</strong> via saved token</div>
    {:else if connected}
      <div class="connected-badge">&#10003; Connected via <code>SLACK_BOT_TOKEN</code></div>
    {/if}
    <details class="token-help">
      <summary>How to get a Slack bot token</summary>
      <ol>
        <li>Go to <a href="https://api.slack.com/apps" target="_blank" rel="noopener">api.slack.com/apps</a> → <strong>Create New App → From scratch</strong></li>
        <li>Under <strong>OAuth &amp; Permissions</strong>, add Bot Token Scopes: <code>channels:history</code>, <code>channels:read</code></li>
        <li>Install the app to your workspace</li>
        <li>Copy the Bot User OAuth Token and paste it below</li>
      </ol>
      <p class="token-note">You can also set <code>SLACK_BOT_TOKEN</code> before starting the API.</p>
    </details>
    <label>
      <span>Slack token <span class="optional">(optional when env token is set)</span></span>
      <input type="password" bind:value={token} placeholder="xoxb-..." />
    </label>
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

  <label style="margin-top:0.75rem">
    <span>URI</span>
    <input type="text" bind:value={uri} placeholder="slack://C1234567890" />
  </label>

  <button on:click={runIngest} disabled={loading || !uri.trim()}>
    {loading ? `Ingesting… (${elapsed}s)` : "Run ingest"}
  </button>

  {#if provider === "codex" && (liveLog || loading)}
    <pre class="live-log">{liveLog || "Waiting for Codex output…"}</pre>
  {/if}

  {#if errorMessage}
    <div class="error">{errorMessage}</div>
  {/if}

  {#if result}
    <ResultPanel {result} {provider} />
  {/if}
</section>

<style>
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

  .hint {
    color: #6b7280;
    font-size: 0.85rem;
    margin: 0 0 1rem;
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
    font-weight: normal;
  }

  .mode-toggle button.active {
    background: #111827;
    color: white;
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

  .live-log {
    margin-top: 0.75rem;
    padding: 0.6rem 0.8rem;
    background: #111827;
    color: #d1fae5;
    border-radius: 6px;
    font-size: 0.78rem;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 200px;
    overflow-y: auto;
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

  code {
    background: #f3f4f6;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
  }
</style>
