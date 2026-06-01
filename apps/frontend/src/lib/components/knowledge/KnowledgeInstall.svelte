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
        codexPlugin: string;
        uriPlaceholder: string;
        uriHint: string;
    }[] = [
        {
            connector: "github",
            label: "GitHub",
            codexPlugin: "github@openai-curated",
            uriPlaceholder: "owner/repo",
            uriHint: "e.g. acme/backend-api",
        },
        {
            connector: "jira",
            label: "Jira",
            codexPlugin: "atlassian-rovo@openai-curated",
            uriPlaceholder: "https://acme.atlassian.net/project/ABC",
            uriHint: "Your Jira project URL",
        },
        {
            connector: "slack",
            label: "Slack",
            codexPlugin: "slack@openai-curated",
            uriPlaceholder: "#channel-name",
            uriHint: "Channel name or conversation ID",
        },
        {
            connector: "notion",
            label: "Notion",
            codexPlugin: "notion@openai-curated",
            uriPlaceholder: "https://notion.so/page-id",
            uriHint: "Notion page or database URL",
        },
        {
            connector: "sharepoint",
            label: "SharePoint / OneDrive",
            codexPlugin: "sharepoint@openai-curated",
            uriPlaceholder: "https://tenant.sharepoint.com/sites/project",
            uriHint: "SharePoint site or OneDrive folder URL",
        },
        {
            connector: "googledrive",
            label: "Google Drive",
            codexPlugin: "google-drive@openai-curated",
            uriPlaceholder: "https://drive.google.com/drive/folders/id",
            uriHint: "Google Drive folder URL or ID",
        },
        {
            connector: "filesystem",
            label: "Filesystem",
            codexPlugin: "",
            uriPlaceholder: "docs/",
            uriHint: "Local path relative to project root",
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
                <div
                    class="connector-row"
                    class:disabled={!pluginReady && r.codexPlugin !== ""}
                >
                    <label class="toggle">
                        <input
                            type="checkbox"
                            bind:checked={enabled[r.connector]}
                            disabled={!pluginReady && r.codexPlugin !== ""}
                        />
                        <span class="conn-label">
                            {statusIcon(st)}
                            {r.label}
                            {#if r.codexPlugin && !pluginReady}
                                <span class="pill error"
                                    >plugin not installed</span
                                >
                            {:else if r.codexPlugin && pluginReady}
                                <span class="pill ok">Codex ready</span>
                            {:else}
                                <span class="pill neutral">direct</span>
                            {/if}
                        </span>
                    </label>

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
    }

    .connector-row {
        border: 1px solid #e5e7eb;
        border-radius: 0.5rem;
        padding: 0.75rem;
        margin-bottom: 0.5rem;
        transition: border-color 0.1s;
    }
    .connector-row.disabled {
        opacity: 0.55;
    }

    .toggle {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        cursor: pointer;
        font-size: 0.9rem;
        font-weight: 500;
    }
    .conn-label {
        display: flex;
        align-items: center;
        gap: 0.4rem;
    }

    .pill {
        font-size: 0.7rem;
        padding: 1px 6px;
        border-radius: 99px;
        font-weight: 400;
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
