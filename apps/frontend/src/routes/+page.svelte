<script lang="ts">
  import { onMount } from "svelte";
  import type { ServiceStatus, CodexPlugin } from "$lib/types";
  import { API_URL, probeService, streamCodexLogin } from "$lib/api";
  import StatusSection from "$lib/components/StatusSection.svelte";
  import GitHubConnector from "$lib/components/GitHubConnector.svelte";
  import SlackConnector from "$lib/components/SlackConnector.svelte";

  const WORKER_URL = "/worker";

  let apiStatus: ServiceStatus = "checking";
  let workerStatus: ServiceStatus = "checking";

  onMount(async () => {
    [apiStatus, workerStatus] = await Promise.all([probeService(API_URL), probeService(WORKER_URL)]);
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
      await streamCodexLogin((line) => {
        codexLoginLog += line + "\n";
      });
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
