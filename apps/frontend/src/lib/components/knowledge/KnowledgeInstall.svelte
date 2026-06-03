<script lang="ts">
    /**
     * KnowledgeInstall — guided first-run flow to ingest all required connectors.
     * Shows readiness per connector, streams ingest progress, marks knowledge ready.
     */
    import { createEventDispatcher } from "svelte";
    import type { CodexPlugin, CodexSourceOption, ConnectorKind } from "$lib/types";
    import {
        getCodexSources,
        getWorkspaceStatus,
        getWorkspaces,
        postFindings,
        resetWorkspace,
    } from "$lib/api";
    import {
        clearAllLocalWorkspaceState,
        setConnectorKnowledge,
        markKnowledgeInstalled,
        project,
    } from "$lib/projectStore";

    export let codexLoggedIn: boolean;
    export let codexPlugins: CodexPlugin[];
    export let embedded = false;
    /** Called when user dismisses the panel. */
    export let onClose: () => void = () => {};

    const dispatch = createEventDispatcher<{ done: void }>();

    // Required connectors for v1 knowledge.
    const REQUIRED: {
        connector: ConnectorKind;
        label: string;
        description: string;
        codexPlugin: string;
        uriPlaceholder: string;
        uriHint: string;
        color: string;
        icon: string;
    }[] = [
        {
            connector: "github",
            label: "GitHub",
            description: "Issues, PRs, commits, and code review discussions",
            codexPlugin: "github@openai-curated",
            uriPlaceholder: "owner/repo",
            uriHint: "e.g. acme/backend-api",
            color: "#24292f",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`,
        },
        {
            connector: "jira",
            label: "Jira",
            description: "Epics, stories, tasks, and sprint boards",
            codexPlugin: "atlassian-rovo@openai-curated",
            uriPlaceholder:
                "https://acme.atlassian.net/jira/software/c/projects/ABC",
            uriHint: "Your Jira project URL",
            color: "#0052CC",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M11.975 0C5.369 0 0 5.369 0 11.975S5.369 23.95 11.975 23.95 23.95 18.581 23.95 11.975 18.581 0 11.975 0zm-.01 4.86l6.307 6.898-6.307 6.898-6.307-6.898 6.307-6.898z"/></svg>`,
        },
        {
            connector: "slack",
            label: "Slack",
            description: "Channel messages, threads, and decisions",
            codexPlugin: "slack@openai-curated",
            uriPlaceholder: "#channel-name",
            uriHint: "Channel name or conversation ID",
            color: "#4A154B",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zm10.122 2.521a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zm-1.268 0a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zm-2.523 10.122a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zm0-1.268a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z"/></svg>`,
        },
        {
            connector: "notion",
            label: "Notion",
            description: "Docs, wikis, databases, and meeting notes",
            codexPlugin: "notion@openai-curated",
            uriPlaceholder: "https://notion.so/page-id",
            uriHint: "Notion page or database URL",
            color: "#000000",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M4.459 4.208c.746.606 1.026.56 2.428.466l13.215-.793c.28 0 .047-.28-.046-.326L17.86 1.968c-.42-.326-.981-.7-2.055-.607L3.01 2.295c-.466.046-.56.28-.374.466zm.793 3.08v13.904c0 .747.373 1.027 1.214.98l14.523-.84c.841-.046.935-.56.935-1.167V6.354c0-.606-.233-.933-.748-.887l-15.177.887c-.56.047-.747.327-.747.933zm14.337.745c.093.42 0 .84-.42.888l-.7.14v10.264c-.608.327-1.168.514-1.635.514-.748 0-.935-.234-1.495-.933l-4.577-7.186v6.952L12.21 19s0 .84-1.168.84l-3.222.186c-.093-.186 0-.653.327-.746l.84-.233V9.854L7.822 9.76c-.094-.42.14-1.026.793-1.073l3.456-.233 4.764 7.279v-6.44l-1.215-.139c-.093-.514.28-.887.747-.933zM1.936 1.035l13.31-.98c1.634-.14 2.055-.047 3.082.7l4.249 2.986c.7.513.934.653.934 1.213v16.378c0 1.026-.373 1.634-1.68 1.726l-15.458.934c-.98.047-1.448-.093-1.962-.747l-3.129-4.06c-.56-.747-.793-1.306-.793-1.96V2.667c0-.839.374-1.54 1.447-1.632z"/></svg>`,
        },
        {
            connector: "sharepoint",
            label: "SharePoint / OneDrive",
            description: "Documents, sites, and team file libraries",
            codexPlugin: "sharepoint@openai-curated",
            uriPlaceholder: "https://tenant.sharepoint.com/sites/project",
            uriHint: "SharePoint site or OneDrive folder URL",
            color: "#0078D4",
            icon: `<span class="text-icon">SP</span>`,
        },
        {
            connector: "googledrive",
            label: "Google Drive",
            description: "Docs, sheets, slides, and shared folders",
            codexPlugin: "google-drive@openai-curated",
            uriPlaceholder: "https://drive.google.com/drive/folders/id",
            uriHint: "Google Drive folder URL or ID",
            color: "#4285F4",
            icon: `<span class="text-icon">GD</span>`,
        },
        {
            connector: "filesystem",
            label: "Filesystem",
            description: "Local docs, markdown files, and OpenAPI specs",
            codexPlugin: "",
            uriPlaceholder: "docs/",
            uriHint: "Local path relative to project root",
            color: "#6b7280",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>`,
        },
    ];

    // Per-connector URI inputs and state.
    let uris: Record<ConnectorKind, string> = {} as Record<
        ConnectorKind,
        string
    >;
    let logs: Record<ConnectorKind, string> = {} as Record<
        ConnectorKind,
        string
    >;
    let statuses: Record<ConnectorKind, "idle" | "running" | "done" | "error"> =
        {} as Record<ConnectorKind, "idle" | "running" | "done" | "error">;
    let enabled: Record<ConnectorKind, boolean> = {} as Record<
        ConnectorKind,
        boolean
    >;
    let sourceOptions: Partial<Record<ConnectorKind, CodexSourceOption[]>> = {};
    let sourceLoading: Partial<Record<ConnectorKind, boolean>> = {};
    let sourceErrors: Partial<Record<ConnectorKind, string>> = {};
    let selectedSources: Partial<Record<ConnectorKind, Record<string, boolean>>> =
        {};

    for (const r of REQUIRED) {
        uris[r.connector] = "";
        logs[r.connector] = "";
        statuses[r.connector] = "idle";
        enabled[r.connector] = false;
    }

    function supportsDiscovery(
        connector: ConnectorKind,
    ): connector is "github" | "slack" {
        return connector === "github" || connector === "slack";
    }

    // Pre-fill URIs from already-ingested connectors in project store.
    $: {
        for (const ck of $project.connectors) {
            if (uris[ck.connector] === "" && ck.uri) {
                uris[ck.connector] = ck.uri;
            }
        }
    }

    function isPluginReady(pluginName: string): boolean {
        if (!pluginName) return true; // filesystem has no plugin
        return (
            codexLoggedIn &&
            codexPlugins.some((p) => p.name === pluginName && p.installed)
        );
    }

    let installing = false;
    let allDone = false;
    let resettingAll = false;

    async function toggleConnector(connector: ConnectorKind, checked: boolean) {
        enabled[connector] = checked;
        enabled = { ...enabled };
    }

    async function loadSourceOptions(connector: "github" | "slack") {
        if (sourceLoading[connector]) return;
        sourceLoading = { ...sourceLoading, [connector]: true };
        sourceErrors = { ...sourceErrors, [connector]: "" };
        try {
            const result = await getCodexSources(connector);
            if (!result) {
                sourceErrors = {
                    ...sourceErrors,
                    [connector]: "Could not load sources from Codex.",
                };
                return;
            }
            sourceOptions = { ...sourceOptions, [connector]: result.sources };
            selectedSources = {
                ...selectedSources,
                [connector]: selectedSources[connector] ?? {},
            };
        } finally {
            sourceLoading = { ...sourceLoading, [connector]: false };
        }
    }

    function toggleSource(connector: ConnectorKind, uri: string, checked: boolean) {
        selectedSources = {
            ...selectedSources,
            [connector]: {
                ...(selectedSources[connector] ?? {}),
                [uri]: checked,
            },
        };
    }

    function selectedTargets() {
        const targets: { connector: ConnectorKind; uri: string }[] = [];
        for (const r of REQUIRED) {
            const manualURI = uris[r.connector].trim();
            if (!enabled[r.connector] && !manualURI) continue;
            const selected = selectedSources[r.connector] ?? {};
            for (const option of sourceOptions[r.connector] ?? []) {
                if (selected[option.uri]) {
                    targets.push({ connector: r.connector, uri: option.uri });
                }
            }
            if (manualURI) {
                targets.push({ connector: r.connector, uri: manualURI });
            }
        }
        const seen = new Set<string>();
        return targets.filter((target) => {
            const key = `${target.connector}:${target.uri}`;
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
        });
    }

    async function installAll() {
        installing = true;
        allDone = false;
        let completed = 0;
        const toRun = selectedTargets();

        for (const target of toRun) {
            const config = REQUIRED.find((r) => r.connector === target.connector);
            if (!config) continue;
            statuses[target.connector] = "running";
            logs[target.connector] =
                (logs[target.connector] || "") +
                `Persisting ${target.uri} into the local workspace database...\n`;
            setConnectorKnowledge(target.connector, target.uri, "ingesting");

            try {
                const res = await postFindings({
                    workspace_id: $project.workspacePath,
                    connector: target.connector,
                    uri: target.uri,
                    provider: config.codexPlugin ? "codex" : "token",
                    role: "pmo",
                    include_execution: false,
                    force_refresh: true,
                });
                if (!res.ok) {
                    const msg =
                        res.body?.message ??
                        res.body?.error ??
                        `Request failed with status ${res.status}`;
                    logs[target.connector] += "[error] " + msg + "\n";
                    statuses[target.connector] = "error";
                    setConnectorKnowledge(target.connector, target.uri, "error", {
                        error: msg,
                    });
                    continue;
                }

                const dbStatus = await getWorkspaceStatus($project.workspacePath);
                const sync = dbStatus?.syncs?.find(
                    (item) =>
                        item.connector === target.connector &&
                        (!item.source_uri || item.source_uri === target.uri) &&
                        (item.event_count ?? 0) > 0,
                );

                const eventCount = sync?.event_count ?? res.body.event_count ?? 0;
                completed += 1;
                statuses[target.connector] = "done";
                logs[target.connector] += sync
                    ? `DB confirmed ${eventCount} persisted event(s), ${res.body.mismatch_count ?? 0} finding(s).\n`
                    : `Saved source. ${eventCount} event(s), ${res.body.mismatch_count ?? 0} finding(s).\n`;
                setConnectorKnowledge(target.connector, target.uri, "ready", {
                    eventCount,
                });
            } catch (e) {
                statuses[target.connector] = "error";
                logs[target.connector] += String(e);
                setConnectorKnowledge(target.connector, target.uri, "error", {
                    error: String(e),
                });
            }
        }

        allDone = toRun.length > 0 && completed === toRun.length;
        if (allDone) {
            markKnowledgeInstalled();
            dispatch("done");
        }
        installing = false;
    }

    async function resetAllData() {
        const confirmed = confirm(
            "Reset all workspace data? This clears saved sources, chat history, graph data, findings, and local workspace memory for every workspace. This cannot be undone.",
        );
        if (!confirmed) return;

        resettingAll = true;
        try {
            const workspaces = await getWorkspaces();
            const paths = new Set<string>([
                $project.workspacePath,
                ...workspaces.map((workspace) => workspace.path),
            ]);
            for (const path of paths) {
                const name =
                    workspaces.find((workspace) => workspace.path === path)?.name ??
                    path.split("/").filter(Boolean).pop() ??
                    "workspace";
                await resetWorkspace(path, name);
            }
            clearAllLocalWorkspaceState([...paths]);
            for (const r of REQUIRED) {
                uris[r.connector] = "";
                logs[r.connector] = "";
                statuses[r.connector] = "idle";
                enabled[r.connector] = false;
            }
            uris = { ...uris };
            logs = { ...logs };
            statuses = { ...statuses };
            enabled = { ...enabled };
            selectedSources = {};
            sourceOptions = {};
            allDone = false;
            dispatch("done");
        } finally {
            resettingAll = false;
        }
    }

    $: selectedCount = selectedTargets().length;
    $: anyEnabled = selectedCount > 0;

    function statusIcon(s: "idle" | "running" | "done" | "error") {
        if (s === "done") return "ready";
        if (s === "running") return "running";
        if (s === "error") return "error";
        return "";
    }
</script>

<div class={embedded ? "ki-inline" : "ki-overlay"}>
    <div class="ki-panel" class:inline={embedded}>
        <div class="ki-header">
            <h2>Workspace Sources</h2>
            <p class="subtitle">
                {$project.name} · {$project.workspacePath}
            </p>
            <button class="close-btn" on:click={onClose} aria-label="Close"
                >Close</button
            >
        </div>

        {#if !codexLoggedIn}
            <div class="warn-banner">
                Codex CLI is not logged in. Run <code
                    >codex login --device-auth</code
                > in your terminal, then reload this page to unlock all connectors.
            </div>
        {/if}

        <div class="connectors-list">
            {#each REQUIRED as r}
                {@const pluginReady = isPluginReady(r.codexPlugin)}
                {@const st = statuses[r.connector]}
                {@const isDisabled = !pluginReady && r.codexPlugin !== ""}
                <label
                    class="connector-card"
                    class:disabled={isDisabled}
                    class:checked={enabled[r.connector]}
                >
                    <!-- brand icon -->
                    <div
                        class="brand-icon"
                        style="background: {r.color}15; color: {r.color};"
                    >
                        {@html r.icon}
                    </div>

                    <!-- info + input -->
                    <div class="card-body">
                        <div class="card-top">
                            <div class="card-title">
                                <span class="conn-name">{r.label}</span>
                                {#if r.codexPlugin && !pluginReady}
                                    <span class="pill error"
                                        >plugin not installed</span
                                    >
                                {:else if r.codexPlugin && pluginReady}
                                    <span class="pill ok">Codex ready</span>
                                {:else}
                                    <span class="pill neutral">direct</span>
                                {/if}
                                {#if st !== "idle"}
                                    <span class="status-icon"
                                        >{statusIcon(st)}</span
                                    >
                                {/if}
                            </div>
                            <p class="conn-desc">{r.description}</p>
                        </div>

                        {#if enabled[r.connector]}
                            {#if supportsDiscovery(r.connector)}
                                <div class="source-picker">
                                    {#if sourceLoading[r.connector]}
                                        <span class="hint">Loading sources from Codex...</span>
                                    {:else if sourceErrors[r.connector]}
                                        <span class="hint error-text">{sourceErrors[r.connector]}</span>
                                        <button
                                            class="mini-btn"
                                            type="button"
                                            on:click|stopPropagation={() => {
                                                if (supportsDiscovery(r.connector)) {
                                                    loadSourceOptions(r.connector);
                                                }
                                            }}
                                        >
                                            Retry
                                        </button>
                                    {:else if sourceOptions[r.connector]?.length}
                                        {#each sourceOptions[r.connector] ?? [] as option}
                                            <label class="source-option">
                                                <input
                                                    type="checkbox"
                                                    checked={Boolean(selectedSources[r.connector]?.[option.uri])}
                                                    on:change={(event) =>
                                                        toggleSource(
                                                            r.connector,
                                                            option.uri,
                                                            (event.currentTarget as HTMLInputElement).checked,
                                                        )}
                                                />
                                                <span>
                                                    <strong>{option.label}</strong>
                                                    <small>{option.kind}</small>
                                                </span>
                                            </label>
                                        {/each}
                                    {:else}
                                        <button
                                            class="mini-btn"
                                            type="button"
                                            on:click|stopPropagation={() => {
                                                if (supportsDiscovery(r.connector)) {
                                                    loadSourceOptions(r.connector);
                                                }
                                            }}
                                        >
                                            Load sources from Codex
                                        </button>
                                        <span class="hint">Optional. You can paste a source manually below.</span>
                                    {/if}
                                </div>

                                <div class="uri-row">
                                    <input
                                        class="uri-input"
                                        type="text"
                                        placeholder={r.uriPlaceholder}
                                        bind:value={uris[r.connector]}
                                    />
                                    <span class="hint">Or enter a source manually: {r.uriHint}</span>
                                </div>
                            {:else}
                                <div class="uri-row">
                                    <input
                                        class="uri-input"
                                        type="text"
                                        placeholder={r.uriPlaceholder}
                                        bind:value={uris[r.connector]}
                                    />
                                    <span class="hint">{r.uriHint}</span>
                                </div>
                            {/if}

                            {#if logs[r.connector]}
                                <pre class="log">{logs[r.connector]}</pre>
                            {/if}
                        {/if}
                    </div>

                    <!-- checkbox -->
                    <input
                        class="card-checkbox"
                        type="checkbox"
                        checked={enabled[r.connector]}
                        on:change={(event) =>
                            toggleConnector(
                                r.connector,
                                (event.currentTarget as HTMLInputElement).checked,
                            )}
                        disabled={isDisabled}
                    />
                </label>
            {/each}
        </div>

        <div class="ki-footer">
            {#if allDone}
                <p class="success-msg">
                    Sources saved. You can now chat about this workspace.
                </p>
                <button class="btn primary" on:click={onClose}
                    >Start chatting</button
                >
            {:else}
                <button
                    class="btn primary"
                    on:click={installAll}
                    disabled={installing || !anyEnabled}
                >
                    {installing ? "Saving..." : `Save ${selectedCount} selected source${selectedCount === 1 ? "" : "s"}`}
                </button>
                <button class="btn secondary" on:click={onClose}
                    >Skip for now</button
                >
                <button
                    class="btn danger"
                    on:click={resetAllData}
                    disabled={installing || resettingAll}
                >
                    {resettingAll ? "Resetting..." : "Reset all data"}
                </button>
            {/if}
        </div>
    </div>
</div>

<style>
    .ki-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0 0 0 / 0.45);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
    }

    .ki-inline {
        width: 100%;
        font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
    }

    :global(.text-icon) {
        font: 700 0.78rem "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
        letter-spacing: 0;
    }

    button,
    input {
        font-family: inherit;
    }

    .ki-panel {
        --ki-pad-x: 1rem;
        background: #f8f6ef;
        border-radius: 0.75rem;
        width: min(640px, 95vw);
        max-height: 90vh;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        box-shadow: 0 20px 40px rgba(0 0 0 / 0.14);
        font-family: inherit;
        box-sizing: border-box;
        padding: 0 var(--ki-pad-x) 1rem;
    }
    .ki-panel.inline {
        --ki-pad-x: 0;
        width: 100%;
        max-height: none;
        border-radius: 0;
        border: 0;
        box-shadow: none;
        background: transparent;
    }

    .ki-header {
        position: sticky;
        top: 0;
        background: inherit;
        padding: 1rem 0 0.75rem;
        padding-right: 6rem;
        border-bottom: 1px solid #d7d2c8;
    }
    .ki-header h2 {
        margin: 0 0 0.25rem;
        font-size: 1.2rem;
    }
    .subtitle {
        margin: 0;
        color: #8a8678;
        font-size: 0.78rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .close-btn {
        position: absolute;
        top: 0.9rem;
        right: 0;
        height: 30px;
        border: 0;
        border-bottom: 1px solid #d7d2c8;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        font-size: 0.78rem;
        font-weight: 700;
        font-family: inherit;
        cursor: pointer;
        color: #1c1b18;
        padding: 0 10px;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s;
    }
    .close-btn:hover {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .warn-banner {
        background: #fef3c7;
        color: #92400e;
        margin-top: 0.75rem;
        padding: 0.6rem 0.75rem;
        font-size: 0.85rem;
        border-left: 2px solid #f59e0b;
    }
    .warn-banner code {
        background: #fde68a;
        padding: 0 3px;
        border-radius: 3px;
    }

    .connectors-list {
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0;
    }

    .connector-card {
        display: flex;
        align-items: flex-start;
        gap: 0.875rem;
        border: 0;
        border-bottom: 1px solid #d7d2c8;
        border-radius: 0;
        padding: 0.85rem 14px;
        cursor: pointer;
        transition: background 0.15s;
        position: relative;
        background: transparent;
        font-family: inherit;
        box-sizing: border-box;
    }
    .connector-card:hover:not(.disabled) {
        background: rgba(255, 253, 247, 0.55);
    }
    .connector-card.checked {
        background: rgba(255, 253, 247, 0.72);
    }
    .connector-card.disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .brand-icon {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        margin-top: 1px;
    }

    .card-body {
        flex: 1;
        min-width: 0;
    }
    .card-top {
        margin-bottom: 0;
    }
    .card-title {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        flex-wrap: wrap;
    }
    .conn-name {
        font-size: 0.9rem;
        font-weight: 600;
        color: #111827;
    }
    .conn-desc {
        margin: 2px 0 0;
        font-size: 0.78rem;
        color: #6b7280;
    }
    .status-icon {
        font-size: 0.65rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: #1c1b18;
        border: 1px solid #8a8678;
        border-radius: 999px;
        padding: 1px 6px;
    }

    .card-checkbox {
        width: 18px;
        height: 18px;
        flex-shrink: 0;
        margin: 2px 0 0 auto;
        cursor: pointer;
        accent-color: #2563eb;
    }

    .pill {
        font-size: 0.68rem;
        padding: 1px 6px;
        border-radius: 99px;
        font-weight: 500;
        white-space: nowrap;
    }
    .pill.ok {
        background: #d1fae5;
        color: #065f46;
    }
    .pill.error {
        background: #fee2e2;
        color: #991b1b;
    }
    .pill.neutral {
        background: #e5e1d8;
        color: #625f55;
    }

    .uri-row {
        margin-top: 0.5rem;
    }

    .source-picker {
        margin-top: 0.6rem;
        display: grid;
        gap: 0.35rem;
        max-height: 12rem;
        overflow: auto;
        border-left: 2px solid #d7d2c8;
        background: transparent;
        padding: 0.3rem 0 0.3rem 0.7rem;
    }

    .source-option {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        border-radius: 0;
        padding: 0.35rem 0;
        cursor: pointer;
    }

    .source-option:hover {
        background: transparent;
    }

    .source-option span,
    .source-option strong,
    .source-option small {
        display: block;
        min-width: 0;
    }

    .source-option strong {
        color: #111827;
        font-size: 0.82rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .source-option small {
        color: #9ca3af;
        font-size: 0.68rem;
        text-transform: uppercase;
    }

    .mini-btn {
        height: 30px;
        width: max-content;
        border: 0;
        border-bottom: 1px solid #d1d5db;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        color: #111827;
        padding: 0 10px;
        font-size: 0.75rem;
        font-weight: 700;
        font-family: inherit;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s;
    }

    .mini-btn:hover {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .error-text {
        color: #991b1b;
    }

    .uri-input {
        width: 100%;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        padding: 0.4rem 0.6rem;
        font-size: 0.85rem;
        box-sizing: border-box;
        outline: none;
        font-family: inherit;
    }
    .uri-input:focus {
        border-bottom-color: #1c1b18;
    }
    .hint {
        font-size: 0.75rem;
        color: #9ca3af;
        margin-top: 2px;
        display: block;
    }

    .log {
        margin-top: 0.4rem;
        background: #1c1b18;
        color: #f8f6ef;
        padding: 0.5rem 0.75rem;
        border-radius: 0;
        font-size: 0.72rem;
        max-height: 6rem;
        overflow-y: auto;
        white-space: pre-wrap;
        font-family: inherit;
    }

    .ki-footer {
        position: sticky;
        bottom: 0;
        background: inherit;
        padding: 1rem 0 0;
        border-top: 1px solid #d7d2c8;
        display: flex;
        gap: 0.75rem;
        align-items: center;
        flex-wrap: wrap;
    }

    .success-msg {
        margin: 0;
        color: #065f46;
        font-size: 0.875rem;
        flex: 1;
    }

    .btn {
        min-height: 34px;
        padding: 0 12px;
        border-radius: 0;
        font-size: 0.875rem;
        font-weight: 700;
        border: 0;
        border-bottom: 1px solid #d7d2c8;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        cursor: pointer;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s,
            opacity 0.15s;
        font-family: inherit;
    }
    .btn.primary {
        color: #1c1b18;
    }
    .btn.primary:hover:not(:disabled) {
        background-position: 0 0;
        border-bottom-color: #1c1b18;
        color: #f8f6ef;
    }
    .btn.primary:disabled {
        color: #8a8678;
        cursor: not-allowed;
        opacity: 0.42;
    }
    .btn.secondary {
        color: #1c1b18;
    }
    .btn.secondary:hover {
        background-position: 0 0;
        border-bottom-color: #1c1b18;
        color: #f8f6ef;
    }
    .btn.danger {
        margin-left: auto;
        background-image: linear-gradient(90deg, #9b3328 0 50%, transparent 50% 100%);
        border-bottom: 1px solid #d85d3f;
        color: #9b3328;
    }
    .btn.danger:hover:not(:disabled) {
        background-position: 0 0;
        border-bottom-color: #9b3328;
        color: #f8f6ef;
    }
    .btn.danger:disabled {
        cursor: not-allowed;
        opacity: 0.42;
    }
</style>
