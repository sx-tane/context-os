<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { CodexPlugin } from "$lib/types";
  import Button from "../ui/Button.svelte";

  /** Full plugin identifier, e.g. "github@openai-curated" */
  export let pluginName: string;
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexPlugins: CodexPlugin[];
  /** Short plugin key passed back in the reauth event, e.g. "github" */
  export let reauthRunning: boolean;
  export let reauthPlugin: string;
  export let reauthLog: string;

  const dispatch = createEventDispatcher<{ reauth: string }>();

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

  {#if plugin?.installed}
    <span style="margin-left:auto">
      <Button
        variant="ghost"
        disabled={reauthRunning && reauthPlugin === shortName}
        on:click={() => dispatch("reauth", shortName)}
        title="Remove and re-add the plugin to connect a different account"
      >
        {reauthRunning && reauthPlugin === shortName
          ? "Re-authing…"
          : `Re-auth ${shortName} plugin`}
      </Button>
    </span>
  {/if}
</div>

{#if reauthLog && reauthPlugin === shortName}
  <pre class="reauth-log">{reauthLog.trim()}</pre>
{/if}

<style>
  .codex-account-badge {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0 0.75rem;
    padding: 0.6rem 0.9rem;
    background: #f0fdf4;
    border: 1px solid #bbf7d0;
    border-radius: 6px;
    font-size: 0.85rem;
    flex-wrap: wrap;
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

  .reauth-log {
    margin: 0.4rem 0 0.75rem;
    padding: 0.6rem 0.8rem;
    background: #111827;
    color: #d1fae5;
    border-radius: 6px;
    font-size: 0.78rem;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 180px;
    overflow-y: auto;
  }

  code {
    background: #f3f4f6;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
  }
</style>
