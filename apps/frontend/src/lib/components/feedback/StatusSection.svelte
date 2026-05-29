<script lang="ts">
  import type { ServiceStatus, CodexPlugin } from "$lib/types";
  import Button from "../ui/Button.svelte";

  export let apiStatus: ServiceStatus;
  export let workerStatus: ServiceStatus;
  export let codexInstalled: boolean;
  export let codexVersion: string;
  export let codexLoggedIn: boolean;
  export let codexAccount: string;
  export let codexLoginLog: string;
  export let codexLoginRunning: boolean;

  export let onLoginClick: () => void;

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
</script>

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

  <div class="row">
    <span class="dot" style="background:{codexLoggedIn ? '#22c55e' : '#888'}"
    ></span>
    <span class="service">Codex CLI</span>
    <span class="value" style="color:{codexLoggedIn ? '#22c55e' : '#888'}">
      {#if codexInstalled}
        {codexLoggedIn ? codexAccount || "Logged in" : "Not logged in"}
      {:else}
        Not installed
      {/if}
      {#if codexVersion}
        <span class="optional">v{codexVersion}</span>
      {/if}
    </span>
    {#if codexInstalled}
      <Button
        variant="ghost"
        disabled={codexLoginRunning}
        on:click={onLoginClick}
        title="Run codex login --device-auth"
      >
        {codexLoginRunning
          ? "Logging in…"
          : codexLoggedIn
            ? "Switch account"
            : "Login"}
      </Button>
    {/if}
  </div>

  {#if codexLoginLog}
    <pre class="login-log">{codexLoginLog.trim()}</pre>
  {/if}
</section>

<style>
  .status {
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

  .optional {
    font-weight: 400;
    color: #9ca3af;
    font-size: 0.8rem;
  }

  .login-log {
    margin: 0.4rem 0 0;
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
</style>
