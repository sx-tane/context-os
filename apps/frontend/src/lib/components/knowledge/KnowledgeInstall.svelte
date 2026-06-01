<script lang="ts">
    /**
     * KnowledgeInstall — guided first-run flow to ingest all required connectors.
     * Shows readiness per connector, streams ingest progress, marks knowledge ready.
     */
    import { createEventDispatcher } from "svelte";
    import type { CodexPlugin, ConnectorKind } from "$lib/types";
    import { runConnectorIngest } from "$lib/ingestRunner";
    import {
        setConnectorKnowledge,
        markKnowledgeInstalled,
        project,
    } from "$lib/projectStore";

    export let codexLoggedIn: boolean;
    export let codexPlugins: CodexPlugin[];
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
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M11.5 0C7.358 0 4 3.358 4 7.5c0 .23.013.457.035.682A4.496 4.496 0 0 0 0 12.5C0 14.985 2.015 17 4.5 17H5v-1.5H4.5C2.843 15.5 1.5 14.157 1.5 12.5c0-1.52 1.11-2.79 2.578-3.025l.595-.094-.127-.592A5.962 5.962 0 0 1 4.5 7.5C4.5 4.467 6.967 2 10 2c2.722 0 5.003 1.922 5.427 4.585l.1.62.627-.032c.115-.005.23-.007.346-.007 2.21 0 4 1.79 4 4s-1.79 4-4 4H16V16h.5c3.038 0 5.5-2.462 5.5-5.5 0-2.894-2.24-5.268-5.083-5.478C16.003 2.147 13.97 0 11.5 0zm.5 10v8.586l-1.793-1.793-1.414 1.414L12 21.414l3.207-3.207-1.414-1.414L12 18.586V10h-1 1z"/></svg>`,
        },
        {
            connector: "googledrive",
            label: "Google Drive",
            description: "Docs, sheets, slides, and shared folders",
            codexPlugin: "google-drive@openai-curated",
            uriPlaceholder: "https://drive.google.com/drive/folders/id",
            uriHint: "Google Drive folder URL or ID",
            color: "#4285F4",
            icon: `<svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M6.28 20.4L2.64 14.01 8.37 4.04 12 10.42 6.28 20.4zm5.43 0H14.4l5.96-10.39H17.6L11.71 20.4zm6.25-11.98l-3.64-6.38h-7.28l3.64 6.38h7.28z"/></svg>`,
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

    for (const r of REQUIRED) {
        uris[r.connector] = "";
        logs[r.connector] = "";
        statuses[r.connector] = "idle";
        enabled[r.connector] = false;
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

    async function installAll() {
        installing = true;
        const toRun = REQUIRED.filter(
            (r) => enabled[r.connector] && uris[r.connector].trim() !== "",
        );

        for (const r of toRun) {
            const uri = uris[r.connector].trim();
            statuses[r.connector] = "running";
            logs[r.connector] = "";
            setConnectorKnowledge(r.connector, uri, "ingesting");

            try {
                await runConnectorIngest({
                    connector: r.connector,
                    uri,
                    provider: r.codexPlugin ? "codex" : "token",
                    setLoading: () => {},
                    setError: (msg) => {
                        if (msg) {
                            logs[r.connector] += "[error] " + msg + "\n";
                            statuses[r.connector] = "error";
                            setConnectorKnowledge(r.connector, uri, "error", {
                                error: msg,
                            });
                        }
                    },
                    setResult: (res) => {
                        if (res) {
                            statuses[r.connector] = "done";
                            setConnectorKnowledge(r.connector, uri, "ready", {
                                eventCount: res.event_count ?? 0,
                            });
                        }
                    },
                    setLiveLog: (update) => {
                        logs[r.connector] =
                            typeof update === "function"
                                ? update(logs[r.connector])
                                : update;
                    },
                    setElapsed: () => {},
                });
            } catch (e) {
                statuses[r.connector] = "error";
                logs[r.connector] += String(e);
                setConnectorKnowledge(r.connector, uri, "error", {
                    error: String(e),
                });
            }
        }

        markKnowledgeInstalled();
        allDone = true;
        installing = false;
        dispatch("done");
    }

    $: anyEnabled = REQUIRED.some(
        (r) => enabled[r.connector] && uris[r.connector].trim() !== "",
    );

    function statusIcon(s: "idle" | "running" | "done" | "error") {
        if (s === "done") return "✅";
        if (s === "running") return "⏳";
        if (s === "error") return "❌";
        return "⬜";
    }
</script>

<div class="ki-overlay">
    <div class="ki-panel">
        <div class="ki-header">
            <h2>Install Project Knowledge</h2>
            <p class="subtitle">
                Connect your data sources. ContextOS will ingest each one and
                build a knowledge graph you can query in chat.
            </p>
            <button class="close-btn" on:click={onClose} aria-label="Close"
                >✕</button
            >
        </div>

        {#if !codexLoggedIn}
            <div class="warn-banner">
                ⚠ Codex CLI is not logged in. Run <code
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
                            <div class="uri-row">
                                <input
                                    class="uri-input"
                                    type="text"
                                    placeholder={r.uriPlaceholder}
                                    bind:value={uris[r.connector]}
                                />
                                <span class="hint">{r.uriHint}</span>
                            </div>

                            {#if logs[r.connector]}
                                <pre class="log">{logs[r.connector]}</pre>
                            {/if}
                        {/if}
                    </div>

                    <!-- checkbox -->
                    <input
                        class="card-checkbox"
                        type="checkbox"
                        bind:checked={enabled[r.connector]}
                        disabled={isDisabled}
                    />
                </label>
            {/each}
        </div>

        <div class="ki-footer">
            {#if allDone}
                <p class="success-msg">
                    ✅ Knowledge installed! You can now chat about your project.
                </p>
                <button class="btn primary" on:click={onClose}
                    >Start chatting →</button
                >
            {:else}
                <button
                    class="btn primary"
                    on:click={installAll}
                    disabled={installing || !anyEnabled}
                >
                    {installing ? "Installing…" : "Install Knowledge"}
                </button>
                <button class="btn secondary" on:click={onClose}
                    >Skip for now</button
                >
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
    }

    .ki-panel {
        background: #fff;
        border-radius: 1rem;
        width: min(640px, 95vw);
        max-height: 90vh;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        box-shadow: 0 20px 40px rgba(0 0 0 / 0.2);
    }

    .ki-header {
        position: sticky;
        top: 0;
        background: #fff;
        padding: 1.25rem 1.5rem 0.75rem;
        border-bottom: 1px solid #f3f4f6;
    }
    .ki-header h2 {
        margin: 0 0 0.25rem;
        font-size: 1.2rem;
    }
    .subtitle {
        margin: 0;
        color: #6b7280;
        font-size: 0.875rem;
    }
    .close-btn {
        position: absolute;
        top: 1rem;
        right: 1rem;
        background: none;
        border: none;
        font-size: 1.1rem;
        cursor: pointer;
        color: #9ca3af;
    }
    .close-btn:hover {
        color: #111827;
    }

    .warn-banner {
        background: #fef3c7;
        color: #92400e;
        padding: 0.6rem 1.5rem;
        font-size: 0.85rem;
        border-bottom: 1px solid #fcd34d;
    }
    .warn-banner code {
        background: #fde68a;
        padding: 0 3px;
        border-radius: 3px;
    }

    .connectors-list {
        padding: 0.75rem 1.5rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .connector-card {
        display: flex;
        align-items: flex-start;
        gap: 0.875rem;
        border: 1.5px solid #e5e7eb;
        border-radius: 0.625rem;
        padding: 0.75rem 0.875rem;
        cursor: pointer;
        transition:
            border-color 0.15s,
            box-shadow 0.15s;
        position: relative;
        background: #fff;
    }
    .connector-card:hover:not(.disabled) {
        border-color: #93c5fd;
        box-shadow: 0 0 0 3px #eff6ff;
    }
    .connector-card.checked {
        border-color: #2563eb;
        background: #f0f7ff;
    }
    .connector-card.disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .brand-icon {
        width: 40px;
        height: 40px;
        border-radius: 0.5rem;
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
        font-size: 0.85rem;
        margin-left: auto;
    }

    .card-checkbox {
        width: 18px;
        height: 18px;
        flex-shrink: 0;
        margin-top: 2px;
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
        background: #e5e7eb;
        color: #6b7280;
    }

    .uri-row {
        margin-top: 0.5rem;
    }
    .uri-input {
        width: 100%;
        border: 1px solid #d1d5db;
        border-radius: 0.375rem;
        padding: 0.4rem 0.6rem;
        font-size: 0.85rem;
        box-sizing: border-box;
        outline: none;
    }
    .uri-input:focus {
        border-color: #2563eb;
    }
    .hint {
        font-size: 0.75rem;
        color: #9ca3af;
        margin-top: 2px;
        display: block;
    }

    .log {
        margin-top: 0.4rem;
        background: #1e1e2e;
        color: #cdd6f4;
        padding: 0.5rem 0.75rem;
        border-radius: 0.375rem;
        font-size: 0.72rem;
        max-height: 6rem;
        overflow-y: auto;
        white-space: pre-wrap;
    }

    .ki-footer {
        position: sticky;
        bottom: 0;
        background: #fff;
        padding: 1rem 1.5rem;
        border-top: 1px solid #f3f4f6;
        display: flex;
        gap: 0.75rem;
        align-items: center;
    }

    .success-msg {
        margin: 0;
        color: #065f46;
        font-size: 0.875rem;
        flex: 1;
    }

    .btn {
        padding: 0.5rem 1.25rem;
        border-radius: 0.5rem;
        font-size: 0.875rem;
        font-weight: 500;
        border: none;
        cursor: pointer;
        transition: background 0.15s;
    }
    .btn.primary {
        background: #2563eb;
        color: #fff;
    }
    .btn.primary:hover:not(:disabled) {
        background: #1d4ed8;
    }
    .btn.primary:disabled {
        background: #93c5fd;
        cursor: not-allowed;
    }
    .btn.secondary {
        background: #f3f4f6;
        color: #374151;
    }
    .btn.secondary:hover {
        background: #e5e7eb;
    }
</style>
