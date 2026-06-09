<script lang="ts">
    import KnowledgeInstall from "$lib/components/knowledge/KnowledgeInstall.svelte";
    import type {
        CodexPlugin,
        ConnectorKnowledge,
        ServiceStatus,
    } from "$lib/types";

    export let codexLabel = "";
    export let workspaceName = "";
    export let readySources: ConnectorKnowledge[] = [];
    export let codexLoggedIn = false;
    export let codexPlugins: CodexPlugin[] = [];
    export let workspaceServiceStatus: ServiceStatus = "checking";
    export let sourcePanelOpen = false;
    export let onClose: () => void = () => {};
    export let onDone: () => void = () => {};
    export let onReset: () => void = () => {};

    $: sourceGroups = groupSources(readySources);

    function groupSources(sources: ConnectorKnowledge[]) {
        const groups = new Map<string, ConnectorKnowledge[]>();
        for (const source of sources) {
            const existing = groups.get(source.connector) ?? [];
            existing.push(source);
            groups.set(source.connector, existing);
        }
        return [...groups.entries()];
    }
</script>

<section class="source-strip">
    <div>
        <span>CODEX</span>
        <strong>{codexLabel}</strong>
    </div>
    <div>
        <span>WORKSPACE</span>
        <strong>{workspaceName}</strong>
    </div>
    <div>
        <span>SOURCES</span>
        <strong>{readySources.length}</strong>
        <small>
            {workspaceServiceStatus === "ok"
                ? codexLoggedIn
                    ? "Codex connected"
                    : "Codex login needed"
                : workspaceServiceStatus === "checking"
                  ? "Local DB checking"
                  : "Local DB unavailable"}
        </small>
    </div>
</section>

{#if sourcePanelOpen}
    <section class="setup-panel">
        <KnowledgeInstall
            embedded
            {codexLoggedIn}
            {codexPlugins}
            workspaceAvailable={workspaceServiceStatus === "ok"}
            {onClose}
            on:done={onDone}
            on:reset={onReset}
        />
    </section>
{:else if readySources.length > 0}
    <section class="source-summary">
        {#each sourceGroups as [connector, sources]}
            <div>
                <strong>{connector}</strong>
                <span>{sources.length} selected</span>
            </div>
        {/each}
    </section>
{:else}
    <section class="source-summary empty-source-summary">
        <div>
            <strong>No sources</strong>
            <span>Use Sources to connect Codex plugins</span>
        </div>
    </section>
{/if}

<style>
    .source-strip,
    .source-summary {
        letter-spacing: 0.05em;
    }

    .source-strip {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 10px;
        align-items: center;
        border-bottom: 1px solid #d7d2c8;
        padding: 4px 0 12px;
    }

    .source-strip span {
        display: block;
        color: #8a8678;
        font-size: 11px;
        text-transform: uppercase;
    }

    .source-strip strong {
        display: block;
        margin-top: 4px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 13px;
    }

    .source-strip small {
        display: block;
        margin-top: 3px;
        color: #8a8678;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 11px;
    }

    .source-summary {
        display: flex;
        gap: 8px;
        overflow-x: auto;
        border-bottom: 1px solid #d7d2c8;
        padding: 0 0 10px;
    }

    .source-summary div {
        min-width: 130px;
        border-left: 2px solid #d7d2c8;
        padding: 2px 10px;
    }

    .source-summary strong,
    .source-summary span {
        display: block;
    }

    .source-summary strong {
        text-transform: uppercase;
        font-size: 12px;
    }

    .source-summary span {
        margin-top: 4px;
        color: #8a8678;
        font-size: 12px;
    }

    .empty-source-summary div {
        min-width: 100%;
    }

    .setup-panel {
        max-height: none;
        overflow: visible;
        border-bottom: 1px solid #d7d2c8;
        padding: 0 12px 12px;
    }
</style>
