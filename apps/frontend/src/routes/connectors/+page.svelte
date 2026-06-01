<script lang="ts">
    import { onMount } from "svelte";
    import type { ServiceStatus, CodexPlugin } from "$lib/types";
    import { API_URL, probeService, streamCodexLogin } from "$lib/api";
    import { sourceConnectorConfigs } from "$lib/sourceConnectorConfigs";
    import StatusSection from "$lib/components/feedback/StatusSection.svelte";
    import GitHubConnector from "$lib/components/connectors/GitHubConnector.svelte";
    import GoogleDriveConnector from "$lib/components/connectors/GoogleDriveConnector.svelte";
    import NotionConnector from "$lib/components/connectors/NotionConnector.svelte";
    import SharePointConnector from "$lib/components/connectors/SharePointConnector.svelte";
    import JiraConnector from "$lib/components/connectors/JiraConnector.svelte";
    import SlackConnector from "$lib/components/connectors/SlackConnector.svelte";
    import SourceConnector from "$lib/components/connectors/SourceConnector.svelte";

    const WORKER_URL = "/worker";

    let apiStatus: ServiceStatus = "checking";
    let workerStatus: ServiceStatus = "checking";

    onMount(async () => {
        [apiStatus, workerStatus] = await Promise.all([
            probeService(API_URL),
            probeService(WORKER_URL),
        ]);
        await checkCodexStatus();
    });

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
    <title>ContextOS — Connector Debug</title>
</svelte:head>

<main>
    <h1>ContextOS — Connector Debug</h1>
    <p class="nav-link">
        <a href="/">← Back to chat</a> ·
        <a href="/findings">Role-based findings →</a>
    </p>

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

    <JiraConnector
        {codexLoggedIn}
        {codexAccount}
        {codexPlugins}
        refreshCodexStatus={checkCodexStatus}
    />

    <GoogleDriveConnector
        {codexLoggedIn}
        {codexAccount}
        {codexPlugins}
        refreshCodexStatus={checkCodexStatus}
    />

    <NotionConnector
        {codexLoggedIn}
        {codexAccount}
        {codexPlugins}
        refreshCodexStatus={checkCodexStatus}
    />

    <SharePointConnector
        {codexLoggedIn}
        {codexAccount}
        {codexPlugins}
        refreshCodexStatus={checkCodexStatus}
    />

    {#each sourceConnectorConfigs as config}
        <SourceConnector
            connector={config.connector}
            title={config.title}
            description={config.description}
            examples={config.examples ?? []}
            defaultUri={config.defaultUri ?? ""}
            uriLabel={config.uriLabel ?? "URI"}
            uriPlaceholder={config.uriPlaceholder ?? ""}
            submitLabel={config.submitLabel ?? "Run ingest"}
            uploadEnabled={config.uploadEnabled ?? false}
            tokenLabel={config.tokenLabel ?? ""}
            tokenPlaceholder={config.tokenPlaceholder ?? ""}
            contentLabel={config.contentLabel ?? ""}
            contentPlaceholder={config.contentPlaceholder ?? ""}
            metadataFields={config.metadataFields ?? []}
            supportedFormats={config.supportedFormats ?? []}
        />
    {/each}
</main>

<style>
    main {
        font-family: system-ui, sans-serif;
        max-width: 860px;
        margin: 3rem auto;
        padding: 0 1rem;
    }

    h1 {
        font-size: 1.75rem;
        margin-bottom: 1.5rem;
    }

    .nav-link {
        margin-bottom: 1rem;
        color: #374151;
    }
</style>
