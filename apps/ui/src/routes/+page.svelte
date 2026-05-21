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
</main>

<style>
  main {
    font-family: system-ui, sans-serif;
    max-width: 480px;
    margin: 4rem auto;
    padding: 0 1rem;
  }

  h1 {
    font-size: 1.75rem;
    margin-bottom: 2rem;
  }

  .status {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 1.25rem 1.5rem;
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
</style>
