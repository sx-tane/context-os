<script lang="ts">
  import type { CodexPlugin } from "$lib/types";

  /** Full plugin identifier, e.g. "github@openai-curated" */
  export let pluginName: string;
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];

  $: plugin = codexPlugins.find((p) => p.name === pluginName);
  $: shortName = pluginName.split("@")[0];
</script>

<div class="codex-account-badge">
  {#if codexLoggedIn}
    <span class="status-dot green"></span>
    <span
      ><strong>{codexAccount || "Logged in"}</strong><span class="optional">
        via OpenAI</span
      ></span
    >
  {:else}
    <span class="status-dot red"></span>
    <span>Not logged in</span>
  {/if}

  {#if plugin?.enabled}
    <span class="sep">·</span>
    <span class="sub"
      >Plugin: <strong style="color:#16a34a">✓ enabled</strong></span
    >
  {:else if plugin?.installed}
    <span class="sep">·</span>
    <span class="sub warn">Plugin installed but not enabled</span>
  {:else if codexPlugins.length > 0}
    <span class="sep">·</span>
    <span class="sub warn"
      >Plugin not installed — run <code>codex plugin add {pluginName}</code
      ></span
    >
  {/if}
</div>

{#if plugin?.installed}
  <p class="reauth-note">
    To reconnect to a different account, run in your terminal:<br />
    <code
      >codex plugin remove {shortName}@openai-curated && codex plugin add {shortName}@openai-curated</code
    >
  </p>
{/if}

<style>
  .codex-account-badge {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0 0.25rem;
    padding: 0.6rem 0.9rem;
    background: #f0fdf4;
    border: 1px solid #bbf7d0;
    border-radius: 6px;
    font-size: 0.85rem;
    flex-wrap: wrap;
  }

  .reauth-note {
    margin: 0 0 0.75rem;
    padding: 0.5rem 0.9rem;
    font-size: 0.78rem;
    color: #6b7280;
    line-height: 1.6;
  }

  .reauth-note code {
    display: block;
    margin-top: 0.3rem;
    background: #f3f4f6;
    padding: 0.3rem 0.5rem;
    border-radius: 4px;
    font-size: 0.76rem;
    word-break: break-all;
  }

  .status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .status-dot.green {
    background: #22c55e;
  }
  .status-dot.red {
    background: #ef4444;
  }

  .sep {
    color: #9ca3af;
    font-size: 0.9rem;
    flex-shrink: 0;
  }

  .sub {
    font-size: 0.82rem;
    color: #374151;
  }

  .warn {
    color: #b45309;
  }

  code {
    background: #f3f4f6;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
  }
</style>
