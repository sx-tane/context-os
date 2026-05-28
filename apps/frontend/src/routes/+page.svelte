<script lang="ts">
  import { onMount } from "svelte";
  import type { ServiceStatus, CodexPlugin } from "$lib/types";
  import StatusSection from "$lib/components/StatusSection.svelte";
  import GitHubConnector from "$lib/components/GitHubConnector.svelte";
  import SlackConnector from "$lib/components/SlackConnector.svelte";

  const API_URL = "/api";
  const WORKER_URL = "/worker";

  let apiStatus: ServiceStatus = "checking";
  let workerStatus: ServiceStatus = "checking";

  async function probe(url: string): Promise<ServiceStatus> {
    try {
      const res = await fetch(`${url}/health`, { signal: AbortSignal.timeout(3000) });
      return res.ok ? "ok" : "unreachable";
    } catch {
      return "unreachable";
    }
  }

  onMount(async () => {
    [apiStatus, workerStatus] = await Promise.all([probe(API_URL), probe(WORKER_URL)]);
    await checkCodexStatus();
  });

  // Codex CLI status — shared across all connectors
  let codexInstalled = false;
  let codexVersion = "";
  let codexLoggedIn = false;
  let codexAccount = "";
  let codexPlugins: CodexPlugin[] = [];
  let codexLoginLog = "";
  let codexLoginRunning = false;

  async function checkCodexStatus() {
    try {
      const res = await fetch(`${API_URL}/codex/status`);
      if (res.ok) {
        const body = await res.json();
        codexInstalled = body?.installed === true;
        codexVersion = body?.version ?? "";
        codexLoggedIn = body?.logged_in === true;
        codexAccount = body?.account ?? "";
        codexPlugins = body?.plugins ?? [];
      }
    } catch {
      // ignore
    }
  }

  async function runCodexLogin() {
    codexLoginLog = "";
    codexLoginRunning = true;
    try {
      const res = await fetch(`${API_URL}/codex/login`, { method: "POST" });
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
          if (dataLine) codexLoginLog += dataLine.slice(5).trim() + "\n";
        }
      }
    } catch (e) {
      codexLoginLog += String(e) + "\n";
    } finally {
      codexLoginRunning = false;
      await checkCodexStatus();
    }
  }
</script>

<svelte:head>
  <title>ContextOS</title>
</svelte:head>

<main>
  <h1>ContextOS</h1>

  <StatusSection
    {apiStatus}
    {workerStatus}
    {codexInstalled}
    {codexVersion}
    {codexLoggedIn}
    {codexAccount}
    {codexLoginLog}
    {codexLoginRunning}
    onLoginClick={runCodexLogin}
  />

  <GitHubConnector
    {codexLoggedIn}
    {codexAccount}
    {codexPlugins}
    refreshCodexStatus={checkCodexStatus}
  />

  <SlackConnector
    {codexLoggedIn}
    {codexAccount}
    {codexPlugins}
    refreshCodexStatus={checkCodexStatus}
  />
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
</style>
