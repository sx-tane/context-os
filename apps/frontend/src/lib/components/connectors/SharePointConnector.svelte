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
    $: void refreshCodexStatus;

    let uri = "";
    let token = "";
    let tenantID = "";
    let clientID = "";
    let clientSecret = "";
    let provider: IngestProvider = "codex";
    let loading = false;
    let errorMessage = "";
    let result: IngestResult | null = null;
    let liveLog = "";
    let elapsed = 0;
    let ingestController: AbortController | null = null;
    let ingestRunID = 0;

    let connected = false;
    let accessTokenConfigured = false;
    let clientCredentialsConfigured = false;
    let tenantConfigured = false;

    onMount(checkStatus);
    onDestroy(() => {
        ingestController?.abort();
    });

    async function checkStatus() {
        const body = await getJSON<{
            connected?: boolean;
            access_token_configured?: boolean;
            client_credentials_configured?: boolean;
            tenant_configured?: boolean;
        }>("/sharepoint/status");
        connected = body?.connected === true;
        accessTokenConfigured = body?.access_token_configured === true;
        clientCredentialsConfigured =
            body?.client_credentials_configured === true;
        tenantConfigured = body?.tenant_configured === true;
    }

    async function runIngest() {
        ingestController?.abort();
        ingestController = new AbortController();
        const runID = ++ingestRunID;
        await runConnectorIngest({
            connector: "sharepoint",
            workspace_id: $project.workspacePath,
            uri,
            token,
            provider,
            ...(tenantID ? { tenant_id: tenantID } : {}),
            ...(clientID ? { client_id: clientID } : {}),
            ...(clientSecret ? { client_secret: clientSecret } : {}),
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
    title="SharePoint / OneDrive MCP Connector"
    description="Ingest SharePoint and OneDrive files via Microsoft Graph using a direct access token, client credentials, or the SharePoint Codex plugin."
    examples={[
        "sharepoint://sites/TENANT.sharepoint.com,SITE_ID/items/ITEM_ID",
        "sharepoint://drives/DRIVE_ID/items/ITEM_ID",
    ]}
>
    <ModeToggle
        bind:value={provider}
        options={[
            { value: "token", label: "Token / credentials" },
            { value: "codex", label: "Codex SharePoint plugin" },
        ]}
        ariaLabel="SharePoint ingestion provider"
    />

    {#if provider === "token"}
        {#if connected}
            <div class="connector-badge">
                &#10003; SharePoint credentials configured{accessTokenConfigured
                    ? " (access token)"
                    : clientCredentialsConfigured
                      ? " (client credentials)"
                      : ""}
            </div>
        {/if}
        <FormField
            label="Access token"
            optional="(optional — use token or client credentials)"
            type="password"
            bind:value={token}
            placeholder="eyJ0..."
        />
        <FormField
            label="Tenant ID"
            optional="(optional when SHAREPOINT_TENANT_ID env is set)"
            bind:value={tenantID}
            placeholder="00000000-0000-0000-0000-000000000000"
        />
        <FormField
            label="Client ID"
            optional="(optional when SHAREPOINT_CLIENT_ID env is set)"
            bind:value={clientID}
            placeholder="11111111-1111-1111-1111-111111111111"
        />
        <FormField
            label="Client secret"
            optional="(optional when SHAREPOINT_CLIENT_SECRET env is set)"
            type="password"
            bind:value={clientSecret}
            placeholder="secret~value"
        />
    {:else}
        <CodexBadge
            {codexLoggedIn}
            {codexAccount}
            {codexPlugins}
            pluginName="sharepoint@openai-curated"
        />
    {/if}

    <FormField
        label="URI"
        bind:value={uri}
        placeholder="sharepoint://sites/TENANT.sharepoint.com,SITE_ID/items/ITEM_ID"
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
