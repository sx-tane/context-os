<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import type { CodexPlugin, IngestProvider, IngestResult } from "$lib/types";
    import { getJSON } from "$lib/api";
    import { project } from "$lib/projectStore";
    import { runConnectorIngest } from "$lib/ingestRunner";
    import ConnectorCard from "./ConnectorCard.svelte";
    import CodexBadge from "./CodexBadge.svelte";
    import ResultPanel from "../feedback/IngestResult.svelte";
    import Button from "../ui/Button.svelte";
    import FormField from "../ui/FormField.svelte";
    import ModeToggle from "../ui/ModeToggle.svelte";
    import LogPanel from "../feedback/LogPanel.svelte";
    import ErrorPanel from "../feedback/ErrorPanel.svelte";

    export let codexLoggedIn: boolean;
    export let codexAccount: string;
    export let codexPlugins: CodexPlugin[];
    export let refreshCodexStatus: () => Promise<void>;

    let uri = "";
    let token = "";
    let provider: IngestProvider = "codex";
    let loading = false;
    let errorMessage = "";
    let result: IngestResult | null = null;
    let liveLog = "";
    let elapsed = 0;
    let ingestController: AbortController | null = null;
    let ingestRunID = 0;

    let connected = false;
    let tokenConfigured = false;

    onMount(checkStatus);
    onDestroy(() => {
        ingestController?.abort();
    });

    async function checkStatus() {
        const body = await getJSON<{
            connected?: boolean;
            token_configured?: boolean;
        }>("/notion/status");
        connected = body?.connected === true;
        tokenConfigured = body?.token_configured === true;
    }

    async function runIngest() {
        ingestController?.abort();
        ingestController = new AbortController();
        const runID = ++ingestRunID;
        await runConnectorIngest({
            connector: "notion",
            workspace_id: $project.workspacePath,
            uri,
            token,
            provider,
            signal: ingestController.signal,
            isCurrent: () => runID === ingestRunID,
            setLoading: (value) => (loading = value),
            setError: (message) => (errorMessage = message),
            setResult: (value) => (result = value),
            setLiveLog: (value) =>
                (liveLog =
                    typeof value === "function" ? value(liveLog) : value),
            setElapsed: (value) =>
                (elapsed =
                    typeof value === "function" ? value(elapsed) : value),
        });
    }
</script>

<ConnectorCard
    title="Notion MCP Connector"
    description="Ingest Notion pages and databases through direct API auth or the Notion Codex plugin."
    examples={[
        "notion://page/PAGE_ID",
        "notion://database/DATABASE_ID",
        "https://www.notion.so/workspace/Page-Title-PAGE_ID",
    ]}
>
    <ModeToggle
        bind:value={provider}
        options={[
            { value: "token", label: "Token / env" },
            { value: "codex", label: "Codex Notion plugin" },
        ]}
        ariaLabel="Notion ingestion provider"
    />

    {#if provider === "token"}
        {#if connected}
            <div class="connector-badge">&#10003; Notion token configured</div>
        {/if}
        <FormField
            label="Notion integration token"
            optional="(optional when NOTION_TOKEN env is set)"
            type="password"
            bind:value={token}
            placeholder="secret_xxxx..."
        />
    {:else}
        <CodexBadge
            {codexLoggedIn}
            {codexAccount}
            {codexPlugins}
            pluginName="notion@openai-curated"
        />
    {/if}

    <FormField
        label="URI"
        bind:value={uri}
        placeholder="notion://page/PAGE_ID"
        offset
    />

    <Button {loading} disabled={loading || !uri.trim()} on:click={runIngest}>
        {loading ? `Ingesting… (${elapsed}s)` : "Run ingest"}
    </Button>

    <LogPanel log={liveLog} {loading} visible={provider === "codex"} />

    <ErrorPanel message={errorMessage} />

    {#if result}
        <ResultPanel {result} {provider} />
    {/if}
</ConnectorCard>
